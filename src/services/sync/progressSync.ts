import logger from '../logger';
import { logError } from '../../utils/errors';
import { FormattedBook, SyncCache } from '../../types';
import { createAudiobookshelfService } from '../audiobookshelf';
import { createHardcoverService } from '../hardcover';
import { MediaProgress } from '../../audiobookshelfTypes';
import { HardcoverAudiobook } from '../../hardcoverTypes';

type AudiobookshelfService = ReturnType<typeof createAudiobookshelfService>;
type HardcoverService = ReturnType<typeof createHardcoverService>;

export enum ProgressSyncResult {
  UPDATED = 'UPDATED',
  SKIPPED = 'SKIPPED',
  NOT_FOUND = 'NOT_FOUND',
  ERROR = 'ERROR',
}

export async function syncAudiobookProgress({
  absService,
  hardcoverService,
  bookProgress,
  cache,
}: {
  absService: AudiobookshelfService;
  hardcoverService: HardcoverService;
  bookProgress: MediaProgress;
  cache?: SyncCache;
}): Promise<ProgressSyncResult> {
  try {
    logger.info(`Syncing progress for book ID: ${bookProgress.libraryItemId}`);

    let formattedBook: FormattedBook | undefined;
    try {
      if (cache?.audiobookDetails?.has(bookProgress.libraryItemId)) {
        const cachedData = cache.audiobookDetails.get(bookProgress.libraryItemId);

        formattedBook = cachedData?.formattedBook;
        logger.debug(`Using cached book data for progress sync: ${bookProgress.libraryItemId}`);
      } else {
        formattedBook = await absService.formatBookForHardcover(bookProgress);

        if (cache?.audiobookDetails) {
          cache.audiobookDetails.set(bookProgress.libraryItemId, {
            formattedBook,
          });
        }
      }
    } catch (error) {
      logError('Failed to format book data for progress sync', error, {
        libraryItemId: bookProgress.libraryItemId,
      });
      return ProgressSyncResult.ERROR;
    }

    if (!formattedBook) {
      logger.warn(`Formatted book data is undefined for ID: ${bookProgress.libraryItemId}`);
      return ProgressSyncResult.ERROR;
    }

    let currentlyReadingAudiobooks;
    if (cache?.hardcoverAudiobooks) {
      currentlyReadingAudiobooks = cache.hardcoverAudiobooks;
      logger.debug('Using cached Hardcover audiobooks list');
    } else {
      currentlyReadingAudiobooks = await hardcoverService.getCurrentlyReadingAudiobooks();

      if (cache) {
        cache.hardcoverAudiobooks = currentlyReadingAudiobooks;
      }
    }

    const matchingAudiobook = findMatchingAudiobook(formattedBook, currentlyReadingAudiobooks);

    if (!matchingAudiobook) {
      logger.warn(
        `No matching audiobook in progress found on Hardcover for "${formattedBook.book.title}"`
      );
      return ProgressSyncResult.NOT_FOUND;
    }

    logger.info(
      `Found matching audiobook on Hardcover - "${formattedBook.book.title}" (Edition ID: ${matchingAudiobook.edition_id})`
    );

    const result = await updateHardcoverProgress({
      hardcoverService,
      editionId: matchingAudiobook.edition_id,
      userId: cache?.hardcoverUserId,
      audiobooks: currentlyReadingAudiobooks,
      formattedBook,
    });

    if (result === true) {
      return ProgressSyncResult.UPDATED;
    } else if (result === 'skipped') {
      return ProgressSyncResult.SKIPPED;
    } else {
      return ProgressSyncResult.ERROR;
    }
  } catch (error) {
    logError('Error syncing audiobook progress', error, {
      libraryItemId: bookProgress.libraryItemId,
    });
    return ProgressSyncResult.ERROR;
  }
}

function findMatchingAudiobook(
  formattedBook: Readonly<FormattedBook>,
  hardcoverAudiobooks: HardcoverAudiobook[]
): HardcoverAudiobook | null {
  if (!hardcoverAudiobooks || hardcoverAudiobooks.length === 0) {
    return null;
  }

  if (formattedBook.book.isbn) {
    const isbnMatch = hardcoverAudiobooks.find(
      audiobook =>
        audiobook.isbn_10 === formattedBook.book.isbn ||
        audiobook.isbn_13 === formattedBook.book.isbn
    );

    if (isbnMatch) {
      logger.info(`Found audiobook match by ISBN: ${formattedBook.book.isbn}`);
      return isbnMatch;
    }
  }

  if (formattedBook.book.asin) {
    const asinMatch = hardcoverAudiobooks.find(
      audiobook => audiobook.asin === formattedBook.book.asin
    );

    if (asinMatch) {
      logger.info(`Found audiobook match by ASIN: ${formattedBook.book.asin}`);
      return asinMatch;
    }
  }

  const titleMatch = hardcoverAudiobooks.find(
    audiobook => audiobook.title.toLowerCase() === formattedBook.book.title.toLowerCase()
  );

  if (titleMatch) {
    logger.info(`Found audiobook match by title: "${formattedBook.book.title}"`);
    return titleMatch;
  }

  return null;
}

async function updateHardcoverProgress({
  hardcoverService,
  editionId,
  userId,
  audiobooks,
  formattedBook,
}: {
  hardcoverService: HardcoverService;
  editionId: number;
  userId?: string | null;
  audiobooks?: HardcoverAudiobook[];
  formattedBook: FormattedBook;
}): Promise<boolean | 'skipped'> {
  const { isFinished, currentTimeSeconds } = formattedBook.progress;

  try {
    logger.info(
      `Updating Hardcover progress for edition ID: ${editionId} to ${currentTimeSeconds} seconds`
    );

    const roundedProgress = Math.round(currentTimeSeconds);

    const bookReadInfo = await hardcoverService.getBookReadInfo(editionId, userId || undefined);

    if (!bookReadInfo) {
      logger.error(`Cannot update progress without book read info for edition: ${editionId}`);
      return false;
    }

    const currentProgressSeconds =
      bookReadInfo.progressSeconds !== null ? bookReadInfo.progressSeconds : 0;

    if (currentProgressSeconds === roundedProgress) {
      logger.info(
        `Skipping update for edition ID: ${editionId} - progress already at ${roundedProgress} seconds`
      );
      return 'skipped';
    }

    const result = await hardcoverService.updateAudiobookProgress({
      editionId,
      progressSeconds: roundedProgress,
      userId: userId || undefined,
      formattedBook,
      bookReadInfo,
    });

    if (result) {
      logger.info(`Successfully updated progress for edition ID: ${editionId}`);

      if (audiobooks && isFinished) {
        await markBookAsFinished(hardcoverService, editionId, audiobooks);
      }
    } else {
      logger.warn(`Failed to update progress for edition ID: ${editionId}`);
    }

    return result;
  } catch (error) {
    logError('Error updating Hardcover progress', error, {
      editionId,
    });
    return false;
  }
}

async function markBookAsFinished(
  hardcoverService: HardcoverService,
  editionId: number,
  audiobooks: HardcoverAudiobook[]
): Promise<void> {
  const audiobook = audiobooks.find(ab => ab.edition_id === editionId);
  if (!audiobook) return;

  const userBookId = audiobook.user_book_id;

  logger.info(`Book is marked as finished in AudiobookShelf. Marking as finished in Hardcover.`);

  const statusUpdateResult = await hardcoverService.updateBookStatus(userBookId, editionId, 3);

  if (statusUpdateResult) {
    logger.info(`Successfully marked book as finished (Edition ID: ${editionId})`);
  } else {
    logger.warn(`Failed to mark book as finished (Edition ID: ${editionId})`);
  }
}

import logger from '../logger';
import { logError } from '../../utils/errors';
import { FormattedBook, SyncCache } from '../../types';
import { createAudiobookshelfService } from '../audiobookshelf';
import { createHardcoverService } from '../hardcover';
import { MediaProgress } from '../../audiobookshelfTypes';
import { HardcoverAudiobook } from '../../hardcoverTypes';

type AudiobookshelfService = ReturnType<typeof createAudiobookshelfService>;
type HardcoverService = ReturnType<typeof createHardcoverService>;

function findBookInHardcoverCache(
  formattedBook: FormattedBook,
  hardcoverAudiobooks: HardcoverAudiobook[]
): HardcoverAudiobook | null {
  // Try to match by ISBN if available
  if (formattedBook.book.isbn) {
    const isbnNormalized = formattedBook.book.isbn.replace(/-/g, '').trim();
    const matchByIsbn = hardcoverAudiobooks.find(
      hcBook => hcBook.isbn_10 === isbnNormalized || hcBook.isbn_13 === isbnNormalized
    );

    if (matchByIsbn) {
      logger.debug(`Matched book "${formattedBook.book.title}" by ISBN`);
      return matchByIsbn;
    }
  }

  // Try to match by ASIN if available
  if (formattedBook.book.asin) {
    const matchByAsin = hardcoverAudiobooks.find(hcBook => hcBook.asin === formattedBook.book.asin);

    if (matchByAsin) {
      logger.debug(`Matched book "${formattedBook.book.title}" by ASIN`);
      return matchByAsin;
    }
  }

  // Try to match by title (simple normalization for better matching)
  const normalizedTitle = formattedBook.book.title.toLowerCase().trim();
  const matchByTitle = hardcoverAudiobooks.find(
    hcBook => hcBook.title.toLowerCase().trim() === normalizedTitle
  );

  if (matchByTitle) {
    logger.debug(`Matched book "${formattedBook.book.title}" by title`);
    return matchByTitle;
  }

  return null;
}

export async function findBookInHardcover({
  absService,
  hardcoverService,
  bookProgress,
  cache,
}: {
  absService: AudiobookshelfService;
  hardcoverService: HardcoverService;
  bookProgress: Readonly<MediaProgress>;
  cache?: SyncCache;
}): Promise<boolean> {
  try {
    let formattedBook: FormattedBook | undefined;

    try {
      if (cache?.audiobookDetails?.has(bookProgress.libraryItemId)) {
        const cachedData = cache.audiobookDetails.get(bookProgress.libraryItemId);
        formattedBook = cachedData?.formattedBook;
        logger.debug(`Using cached book data for ID: ${bookProgress.libraryItemId}`);
      } else {
        formattedBook = await absService.formatBookForHardcover(bookProgress);

        if (cache?.audiobookDetails) {
          cache.audiobookDetails.set(bookProgress.libraryItemId, {
            formattedBook,
          });
        }
      }
    } catch (error) {
      logError(`Failed to format book data for ID: ${bookProgress.libraryItemId}`, error, {
        libraryItemId: bookProgress.libraryItemId,
      });
      throw error;
    }

    if (!formattedBook) {
      logger.warn(`Formatted book data is undefined for ID: ${bookProgress.libraryItemId}`);
      return false;
    }

    logger.info(`Syncing book "${formattedBook.book.title}" with Hardcover`);

    if (cache?.hardcoverAudiobooks?.length) {
      const cachedMatch = findBookInHardcoverCache(formattedBook, cache.hardcoverAudiobooks);

      if (cachedMatch) {
        logger.info(
          `Found book "${formattedBook.book.title}" in Hardcover cache (Edition ID: ${cachedMatch.edition_id})`
        );
        return true;
      }

      logger.debug(
        `Book "${formattedBook.book.title}" not found in Hardcover cache, falling back to API search`
      );
    }

    try {
      const hardcoverBook = await hardcoverService.findBook(formattedBook.book);

      if (!hardcoverBook) {
        logger.warn(`Book "${formattedBook.book.title}" not found in Hardcover library`);
        return false;
      }

      logger.info(
        `Found book "${formattedBook.book.title}" in Hardcover (ID: ${hardcoverBook.id})`
      );

      return true;
    } catch (error) {
      logError(`Failed to find book "${formattedBook.book.title}" in Hardcover`, error, {
        bookTitle: formattedBook.book.title,
      });
      throw error;
    }
  } catch (error) {
    throw error;
  }
}

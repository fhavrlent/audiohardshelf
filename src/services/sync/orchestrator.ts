import logger from '../logger';
import { logError } from '../../utils/errors';
import { createAudiobookshelfService } from '../audiobookshelf';
import { createHardcoverService } from '../hardcover';
import { findBookInHardcover } from './matcher';
import { syncAudiobookProgress, ProgressSyncResult } from './progressSync';
import { MediaProgress } from '../../audiobookshelfTypes';
import { SyncCache } from '../../types';

type AudiobookshelfService = ReturnType<typeof createAudiobookshelfService>;
type HardcoverService = ReturnType<typeof createHardcoverService>;

type SyncResults = Readonly<{
  notFoundCount: number;
  updatedProgressCount: number;
  skippedProgressCount: number;
  errorCount: number;
}>;

export async function syncBooksToHardcover(
  absService: AudiobookshelfService,
  hardcoverService: HardcoverService
): Promise<void> {
  try {
    logger.info('Starting book sync process between Audiobookshelf and Hardcover');

    const currentlyListening = await getCurrentlyListening(absService);

    if (currentlyListening.length === 0) {
      logger.warn('No books to sync - no currently listening books found');
      return;
    }

    const booksToSync: MediaProgress[] = [...currentlyListening];

    logger.info(`Finding ${booksToSync.length} books in Hardcover for sync`);

    const cache: SyncCache = {
      audiobookDetails: new Map(),
      hardcoverAudiobooks: null,
      hardcoverUserId: null,
    };

    const results = await processBooksSync({
      absService,
      hardcoverService,
      books: booksToSync,
      cache,
    });

    logger.info(
      `Book sync completed. Not Found: ${results.notFoundCount}, ` +
        `Progress Updated: ${results.updatedProgressCount}, Progress Skipped: ${results.skippedProgressCount}, ` +
        `Errors: ${results.errorCount}`
    );
  } catch (error) {
    logError('Error during book sync process', error);
    throw error;
  }
}

async function getCurrentlyListening(absService: AudiobookshelfService): Promise<MediaProgress[]> {
  try {
    const currentlyListening = await absService.getCurrentlyListening();

    logger.info(`Found ${currentlyListening.length} books currently in progress on Audiobookshelf`);
    return currentlyListening;
  } catch (error) {
    logError('Failed to get currently listening books', error);
    return [];
  }
}

async function processBooksSync({
  absService,
  hardcoverService,
  books,
  cache,
}: {
  absService: AudiobookshelfService;
  hardcoverService: HardcoverService;
  books: MediaProgress[];
  cache: SyncCache;
}): Promise<SyncResults> {
  let notFoundCount = 0;
  let updatedProgressCount = 0;
  let skippedProgressCount = 0;
  let errorCount = 0;

  if (!cache.hardcoverUserId) {
    await hardcoverService.validateConnection();
    try {
      const hardcoverAudiobooks = await hardcoverService.getCurrentlyReadingAudiobooks();
      cache.hardcoverAudiobooks = hardcoverAudiobooks;
    } catch (error) {
      logError('Error pre-fetching Hardcover data', error);
    }
  }

  const processedIds = new Set<string>();

  for (const bookProgress of books) {
    try {
      if (processedIds.has(bookProgress.libraryItemId)) {
        logger.debug(`Skipping duplicate book with ID: ${bookProgress.libraryItemId}`);
        continue;
      }

      processedIds.add(bookProgress.libraryItemId);

      const found = await findBookInHardcover({
        absService,
        hardcoverService,
        bookProgress,
        cache,
      });

      if (found) {
        const progressResult = await syncAudiobookProgress({
          absService,
          hardcoverService,
          bookProgress,
          cache,
        });

        switch (progressResult) {
          case ProgressSyncResult.UPDATED:
            updatedProgressCount++;
            break;
          case ProgressSyncResult.SKIPPED:
            skippedProgressCount++;
            break;
          case ProgressSyncResult.NOT_FOUND:
            notFoundCount++;
            break;
          case ProgressSyncResult.ERROR:
            errorCount++;
            break;
        }
      } else {
        notFoundCount++;
      }
    } catch (error) {
      errorCount++;
      logError('Error syncing book', error, {
        libraryItemId: bookProgress.libraryItemId,
      });
    }
  }

  return {
    notFoundCount,
    updatedProgressCount,
    skippedProgressCount,
    errorCount,
  };
}

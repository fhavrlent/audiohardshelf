import { createAudiobookshelfService } from './audiobookshelf';
import { createHardcoverService } from './hardcover';
import { logError } from '../utils/errors';
import { syncBooksToHardcover } from './sync/orchestrator';

export function createSyncService() {
  try {
    const absService = createAudiobookshelfService();
    const hardcoverService = createHardcoverService();

    return {
      findBooksInHardcover: async (): Promise<void> =>
        syncBooksToHardcover(absService, hardcoverService),
    };
  } catch (error) {
    logError('Error initializing sync service', error);
    throw error;
  }
}

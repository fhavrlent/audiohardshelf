import cron from 'node-cron';
import config from './config/config';
import { createSyncService } from './services/sync';
import logger from './services/logger';
import { logError, extractErrorMessage } from './utils/errors';
import fs from 'fs';

let syncService: ReturnType<typeof createSyncService>;

function validateConfiguration(): void {
  const missingConfigs: string[] = [];

  if (!config.audiobookshelf.url) missingConfigs.push('ABS_URL');
  if (!config.audiobookshelf.apiKey) missingConfigs.push('ABS_API_KEY');
  if (!config.audiobookshelf.userId) missingConfigs.push('ABS_USER_ID');
  if (!config.hardcover.apiKey) missingConfigs.push('HARDCOVER_API_KEY');

  if (missingConfigs.length) {
    const errorMessage = `Missing required configuration variables: ${missingConfigs.join(', ')}`;
    logger.error(errorMessage);

    if (!fs.existsSync('.env')) {
      logger.error(
        `The .env file is missing. Please copy .env.example to .env and fill in your configuration.`
      );
    }

    throw new Error(errorMessage);
  }
}

export async function runBookSearch(): Promise<void> {
  try {
    logger.info('Manual sync initiated');

    validateConfiguration();

    if (!syncService) {
      syncService = createSyncService();
    }

    await syncService.findBooksInHardcover();
    logger.info('Manual sync completed');
  } catch (error) {
    logError('Error during manual sync', error);

    const errorMessage = extractErrorMessage(error);
    if (errorMessage.includes('404')) {
      logger.error('API endpoint not found (404)', {
        statusCode: 404,
        currentUrl: config.audiobookshelf.url,
        suggestions: [
          'Verify ABS_URL in .env includes http:// or https:// and correct port',
          'Check if Audiobookshelf server is running',
          'Verify server version compatibility',
          'Check network connectivity',
        ],
        helpDocs: 'See logs directory for full error details',
      });
    }
  }
}

function scheduleSearch(): void {
  try {
    validateConfiguration();

    const syncInterval = config.syncInterval;

    const intervalMinutes = parseInt(syncInterval);

    if (!isNaN(intervalMinutes) && intervalMinutes > 0) {
      //
      logger.info(`Scheduling automatic sync every ${intervalMinutes} minutes from startup`);

      setInterval(
        async () => {
          logger.info('Interval-based sync job starting');
          try {
            if (!syncService) {
              syncService = createSyncService();
            }

            await syncService.findBooksInHardcover();
            logger.info('Interval-based sync job completed successfully');
          } catch (error) {
            logError('Error during interval-based sync job', error);
          }
        },
        intervalMinutes * 60 * 1000
      );
    } else {
      logger.info(`Scheduling automatic sync with cron pattern: ${syncInterval}`);

      cron.schedule(syncInterval, async () => {
        logger.info('Scheduled sync job starting');
        try {
          if (!syncService) {
            syncService = createSyncService();
          }

          await syncService.findBooksInHardcover();
          logger.info('Scheduled sync job completed successfully');
        } catch (error) {
          logError('Error during scheduled sync job', error);
        }
      });
    }

    logger.info('Automatic sync scheduled successfully');
  } catch (error) {
    logError('Failed to schedule automatic sync', error);
    throw error;
  }
}

if (require.main === module) {
  (async () => {
    try {
      logger.info('AudioHardShelf is starting up');

      try {
        validateConfiguration();
      } catch {
        process.exit(1);
      }

      try {
        syncService = createSyncService();

        scheduleSearch();

        logger.info('Running initial sync on startup');
        await runBookSearch();

        logger.info('AudioHardShelf is running. Press Ctrl+C to exit');
      } catch (error) {
        logError('Error during startup', error);
        process.exit(1);
      }
    } catch (error) {
      logError('Error starting AudioHardShelf', error);
      process.exit(1);
    }
  })();
}

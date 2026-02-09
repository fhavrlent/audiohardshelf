import axios from 'axios';
import logger from '../logger';
import { logError } from '../../utils/errors';
import { validateAudiobookshelfConnection } from './audiobookshelfConnection';
import { MediaProgress } from '../../audiobookshelfTypes';
import { AudiobookshelfClient } from '../../types';

export async function getCurrentlyListeningBooks(
  client: AudiobookshelfClient
): Promise<MediaProgress[]> {
  try {
    const apiClient = client.getClient();
    const isConnected = await validateAudiobookshelfConnection(client);
    if (!isConnected) {
      logger.warn('Skipping getCurrentlyListeningBooks due to connection issues');
      return [];
    }

    logger.info('Fetching currently listening books from Audiobookshelf');

    // Fetch user data which includes all media progress
    const response = await apiClient.get<{ mediaProgress: MediaProgress[] }>('/api/me');

    if (!response.data || !response.data.mediaProgress || !Array.isArray(response.data.mediaProgress)) {
      logger.warn('Unexpected response format from /api/me endpoint');
      return [];
    }

    // Filter for books only (exclude podcasts) and in-progress items
    const booksInProgress = response.data.mediaProgress.filter(
      progress =>
        progress &&
        progress.mediaItemType === 'book' &&
        !progress.isFinished &&
        !progress.hideFromContinueListening
    );

    logger.info(`Found ${booksInProgress.length} audiobooks in progress`);

    // Deduplicate by libraryItemId (shouldn't be needed but keeping for safety)
    const uniqueBookIds = new Set<string>();
    const uniqueBooksInProgress = booksInProgress.filter(progress => {
      if (!progress || !progress.libraryItemId) return false;

      if (uniqueBookIds.has(progress.libraryItemId)) {
        logger.debug(`Skipping duplicate book from Audiobookshelf API response: ${progress.libraryItemId}`);
        return false;
      }

      uniqueBookIds.add(progress.libraryItemId);
      return true;
    });

    if (uniqueBooksInProgress.length < booksInProgress.length) {
      logger.warn(
        `Removed ${
          booksInProgress.length - uniqueBooksInProgress.length
        } duplicate books from Audiobookshelf response`
      );
    }

    return uniqueBooksInProgress;
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logError('Error fetching currently listening books', error, {
      statusCode,
      details: JSON.stringify(errorDetails),
      endpoint: '/api/me',
    });

    return [];
  }
}

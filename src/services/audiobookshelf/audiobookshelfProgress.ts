import axios from 'axios';
import logger from '../logger';
import { logError } from '../../utils/errors';
import { validateAudiobookshelfConnection } from './audiobookshelfConnection';
import {
  InProgressLibraryItem,
  ItemsInProgressResponse,
  MediaProgress,
} from '../../audiobookshelfTypes';
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

    const response = await apiClient.get<ItemsInProgressResponse>('/api/me/items-in-progress');

    if (
      !response.data ||
      !response.data.libraryItems ||
      !Array.isArray(response.data.libraryItems)
    ) {
      logger.warn('Unexpected response format from items-in-progress endpoint');
      return [];
    }

    const booksInProgress = response.data.libraryItems.filter(
      item => item && item.mediaType === 'book'
    );

    logger.info(
      `Found ${response.data.libraryItems.length} items in progress, ${booksInProgress.length} of those are audiobooks`
    );

    const uniqueBookIds = new Set<string>();
    const uniqueBooksInProgress = booksInProgress.filter(item => {
      if (!item || !item.id) return false;

      if (uniqueBookIds.has(item.id)) {
        logger.debug(`Skipping duplicate book from Audiobookshelf API response: ${item.id}`);
        return false;
      }

      uniqueBookIds.add(item.id);
      return true;
    });

    if (uniqueBooksInProgress.length < booksInProgress.length) {
      logger.warn(
        `Removed ${
          booksInProgress.length - uniqueBooksInProgress.length
        } duplicate books from Audiobookshelf response`
      );
    }

    const progressPromises = uniqueBooksInProgress.map(async item => {
      if (!item || !item.id) {
        logger.warn('Found invalid item in progress (missing ID), skipping');
        return null;
      }

      try {
        const { data: progressData } = await apiClient.get<MediaProgress>(
          `/api/me/progress/${item.id}`
        );

        if (!progressData) {
          logger.warn(`No progress data found for item ${item.id}`);
          return createAudiobookFallbackProgressData(item);
        }

        return progressData;
      } catch (error) {
        logger.warn(`Error fetching progress for item ${item.id}, using fallback data`, {
          error: error instanceof Error ? error.message : 'Unknown error',
        });
        return createAudiobookFallbackProgressData(item);
      }
    });

    const progressResults = await Promise.all(progressPromises);

    return progressResults.filter(Boolean) as MediaProgress[];
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logError('Error fetching currently listening books', error, {
      statusCode,
      details: JSON.stringify(errorDetails),
      endpoint: '/api/me/items-in-progress',
    });

    return [];
  }
}

function createAudiobookFallbackProgressData(item: InProgressLibraryItem) {
  const mediaMetadata = item.media?.metadata || {};
  const duration =
    ('duration' in mediaMetadata ? mediaMetadata.duration : 0) ||
    (item.media && 'duration' in item.media ? item.media.duration : 0) ||
    0;

  return {
    libraryItemId: item.id,
    currentTime: 0,
    progress: 0,
    isFinished: false,
    lastUpdate: item.progressLastUpdate
      ? new Date(item.progressLastUpdate).toISOString()
      : new Date().toISOString(),
    duration: duration,
  };
}

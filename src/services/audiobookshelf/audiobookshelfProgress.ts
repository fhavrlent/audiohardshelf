import axios from 'axios';
import logger from '../logger';
import { logError } from '../../utils/errors';
import { validateAudiobookshelfConnection } from './audiobookshelfConnection';
import { Library, LibrariesResponse, MediaProgress } from '../../audiobookshelfTypes';
import { AudiobookshelfClient } from '../../types';

async function getAudiobookLibraries(client: AudiobookshelfClient): Promise<Library[]> {
  try {
    const apiClient = client.getClient();
    logger.info('Fetching libraries from Audiobookshelf');

    const response = await apiClient.get<LibrariesResponse>('/api/libraries');

    if (!response.data || !response.data.libraries || !Array.isArray(response.data.libraries)) {
      logger.warn('Unexpected response format from /api/libraries endpoint');
      return [];
    }

    const audiobookLibraries = response.data.libraries.filter(
      lib => lib && lib.mediaType === 'book'
    );

    logger.info(
      `Found ${response.data.libraries.length} total libraries, ${audiobookLibraries.length} audiobook libraries`
    );

    return audiobookLibraries;
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logError('Error fetching audiobook libraries', error, {
      statusCode,
      details: JSON.stringify(errorDetails),
      endpoint: '/api/libraries',
    });

    return [];
  }
}

async function getLibraryItemsInProgress(
  client: AudiobookshelfClient,
  libraryId: string,
  limit: number = 50
): Promise<any[]> {
  try {
    const apiClient = client.getClient();
    logger.debug(`Fetching in-progress items for library ${libraryId}`);

    // Use exact same format as Audiobookshelf UI
    const filterValue = Buffer.from('in-progress').toString('base64');

    const params = new URLSearchParams({
      filter: `progress.${filterValue}`,
      sort: 'media.metadata.title',
      desc: '0',
      limit: limit.toString(),
      page: '0',
    });

    const response = await apiClient.get<{ results: any[]; total: number }>(
      `/api/libraries/${libraryId}/items?${params.toString()}`
    );

    if (!response.data || !response.data.results || !Array.isArray(response.data.results)) {
      logger.warn(`Unexpected response format from /api/libraries/${libraryId}/items endpoint`);
      return [];
    }

    logger.debug(`Found ${response.data.results.length} in-progress items in library ${libraryId}`);

    return response.data.results;
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logError(`Error fetching in-progress items for library ${libraryId}`, error, {
      statusCode,
      details: JSON.stringify(errorDetails),
      endpoint: `/api/libraries/${libraryId}/items`,
      libraryId,
      limit,
    });

    return [];
  }
}

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

    // Step 1: Get list of in-progress item IDs from library endpoint (source of truth)
    const audiobookLibraries = await getAudiobookLibraries(client);

    if (audiobookLibraries.length === 0) {
      logger.warn('No audiobook libraries found');
      return [];
    }

    const itemsPromises = audiobookLibraries.map(library =>
      getLibraryItemsInProgress(client, library.id, 50)
    );

    const itemsArrays = await Promise.all(itemsPromises);
    const allItems = itemsArrays.flat();

    // Get set of valid in-progress item IDs
    const validItemIds = new Set(allItems.map(item => item.id).filter(Boolean));

    logger.info(
      `Found ${validItemIds.size} audiobooks in progress across ${audiobookLibraries.length} libraries`
    );

    // Step 2: Get ALL progress from /api/me
    const response = await apiClient.get<{ mediaProgress: MediaProgress[] }>('/api/me');

    if (
      !response.data ||
      !response.data.mediaProgress ||
      !Array.isArray(response.data.mediaProgress)
    ) {
      logger.warn('Unexpected response format from /api/me endpoint');
      return [];
    }

    // Step 3: Filter progress to ONLY items that are in the library in-progress list
    const filteredProgress = response.data.mediaProgress.filter(
      progress =>
        progress && progress.mediaItemType === 'book' && validItemIds.has(progress.libraryItemId)
    );

    logger.info(`Matched ${filteredProgress.length} progress records to in-progress items`);

    return filteredProgress;
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logError('Error fetching currently listening books', error, {
      statusCode,
      details: JSON.stringify(errorDetails),
      endpoint: 'library-based + /api/me',
    });

    return [];
  }
}

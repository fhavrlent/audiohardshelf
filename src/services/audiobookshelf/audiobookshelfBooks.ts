import axios from 'axios';
import logger from '../logger';
import { logError } from '../../utils/errors';
import { BookExpanded, LibraryItemExpanded } from '../../audiobookshelfTypes';
import { AudiobookshelfClient } from '../../types';

export async function getAudiobookDetails(
  client: AudiobookshelfClient,
  libraryItemId: string
): Promise<LibraryItemExpanded> {
  try {
    const apiClient = client.getClient();
    logger.info(`Fetching audiobook details for ${libraryItemId}`);

    const { data } = await apiClient.get<LibraryItemExpanded>(`/api/items/${libraryItemId}`);
    if (data) {
      return data;
    } else {
      logger.warn(`Empty response from /api/items/${libraryItemId}`);
      return createAudiobookFallbackData(libraryItemId);
    }
  } catch (error) {
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';

    logError(`Error fetching audiobook details for ${libraryItemId}`, error, {
      statusCode,
      endpoint: `/api/items/${libraryItemId}`,
      libraryItemId,
    });

    return createAudiobookFallbackData(libraryItemId);
  }
}

function createAudiobookFallbackData(libraryItemId: string): LibraryItemExpanded {
  return {
    id: '',
    ino: '',
    libraryId: '',
    folderId: '',
    path: '',
    relPath: '',
    isFile: false,
    mtimeMs: 0,
    ctimeMs: 0,
    birthtimeMs: 0,
    addedAt: 0,
    updatedAt: 0,
    lastScan: null,
    scanVersion: null,
    isMissing: false,
    isInvalid: false,
    mediaType: 'book',
    media: {
      libraryItemId,
      metadata: {
        title: null,
        subtitle: null,
        authors: [],
        narrators: [],
        series: [],
        genres: [],
        publishedYear: null,
        publishedDate: null,
        publisher: null,
        description: null,
        isbn: null,
        asin: null,
        language: null,
        explicit: false,
        titleIgnorePrefix: '',
        authorName: '',
        authorNameLF: '',
        narratorName: '',
        seriesName: '',
      },
      coverPath: null,
      tags: [],
      audioFiles: [],
      chapters: [],
      duration: 0,
      size: 0,
      tracks: [],
      ebookFile: null,
    } as BookExpanded,
    libraryFiles: [],
    size: 0,
  };
}

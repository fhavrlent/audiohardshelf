import logger from '../logger';
import { logError } from '../../utils/errors';
import { getAudiobookDetails } from './audiobookshelfBooks';
import { AudiobookshelfClient, FormattedBook } from '../../types';
import {
  BookMetadataExpanded,
  MediaProgress,
  PodcastMetadataExpanded,
} from '../../audiobookshelfTypes';

function formatTimestampToDate(timestamp: number): string {
  const date = new Date(Number(timestamp));
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const day = String(date.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
}

export async function formatAudiobookForHardcover(
  client: AudiobookshelfClient,
  mediaProgress: MediaProgress
): Promise<FormattedBook> {
  try {
    const bookDetails = await getAudiobookDetails(client, mediaProgress.libraryItemId);
    if (!bookDetails) {
      throw new Error(`Book details not found for ID: ${mediaProgress.libraryItemId}`);
    }

    if (!bookDetails.media) {
      logger.warn(
        `Book with ID ${mediaProgress.libraryItemId} has no media property, using fallback data structure`
      );

      return createAudiobookFormatFallback(bookDetails, mediaProgress);
    }

    const metadata = bookDetails.media.metadata || {};
    const authors = extractAudiobookAuthors(metadata);
    const progressPercent = calculateAudiobookProgressPercent(mediaProgress);

    return {
      book: {
        title: metadata.title || bookDetails.media.metadata.title || 'Unknown Title',
        isbn: ('isbn' in bookDetails.media.metadata && bookDetails.media.metadata.isbn) || null,
        authors: authors,
        asin: ('asin' in bookDetails.media.metadata && bookDetails.media.metadata.asin) || null,
      },
      progress: {
        percent: progressPercent || 0,
        currentTimeSeconds: mediaProgress.currentTime || 0,
        durationSeconds:
          mediaProgress.duration ||
          ('duration' in bookDetails.media && bookDetails.media.duration) ||
          0,
        lastUpdate: mediaProgress.lastUpdate.toString() || new Date().toISOString(),
        isFinished: mediaProgress.isFinished || false,
        startedAt: formatTimestampToDate(mediaProgress.startedAt),
        finishedAt: mediaProgress.finishedAt
          ? formatTimestampToDate(mediaProgress.finishedAt)
          : undefined,
      },
    };
  } catch (error) {
    logError('Error formatting audiobook for Hardcover', error, {
      libraryItemId: mediaProgress.libraryItemId,
    });

    return {
      book: {
        title: `Unknown Book (ID: ${mediaProgress.libraryItemId})`,
        isbn: null,
        asin: null,
        authors: ['Unknown Author'],
      },
      progress: {
        percent: mediaProgress.progress || 0,
        currentTimeSeconds: mediaProgress.currentTime || 0,
        durationSeconds: mediaProgress.duration || 0,
        lastUpdate: mediaProgress.lastUpdate.toString() || new Date().toISOString(),
        isFinished: mediaProgress.isFinished || false,
      },
    };
  }
}

function extractAudiobookAuthors(
  metadata: BookMetadataExpanded | PodcastMetadataExpanded
): string[] {
  if ('authors' in metadata && metadata.authors && Array.isArray(metadata.authors)) {
    return metadata.authors
      .filter(author => author && typeof author.name === 'string')
      .map(author => author.name);
  }

  if ('authorName' in metadata && metadata.authorName) {
    return [metadata.authorName];
  }

  if ('author' in metadata && metadata.author) {
    if (typeof metadata.author === 'string') {
      return [metadata.author];
    }

    if (Array.isArray(metadata.author)) {
      const authorArray = metadata.author as string[];
      return authorArray.filter(a => typeof a === 'string');
    }
  }

  return ['Unknown Author'];
}

function calculateAudiobookProgressPercent(mediaProgress: MediaProgress): number {
  if (mediaProgress.progress) {
    return mediaProgress.progress;
  }

  if (mediaProgress.duration && mediaProgress.currentTime) {
    return Math.round((mediaProgress.currentTime / mediaProgress.duration) * 100);
  }

  return 0;
}

function createAudiobookFormatFallback(
  bookDetails: unknown,
  mediaProgress: MediaProgress
): FormattedBook {
  const details = bookDetails as Record<string, unknown>;

  const fallbackTitle =
    (typeof details.title === 'string' ? details.title : null) ||
    (typeof details.name === 'string' ? details.name : null) ||
    'Unknown Title';

  const fallbackAuthor =
    (typeof details.author === 'string' ? details.author : null) ||
    (typeof details.authorName === 'string' ? details.authorName : null) ||
    'Unknown Author';

  const fallbackDuration = mediaProgress.duration || 0;

  const authors = Array.isArray(fallbackAuthor) ? [...fallbackAuthor] : [fallbackAuthor];

  return {
    book: {
      title: fallbackTitle,
      isbn: typeof details.isbn === 'string' ? details.isbn : null,
      asin: typeof details.asin === 'string' ? details.asin : null,
      authors: authors,
    },
    progress: {
      percent: mediaProgress.progress || 0,
      currentTimeSeconds: mediaProgress.currentTime || 0,
      durationSeconds: fallbackDuration,
      lastUpdate: mediaProgress.lastUpdate.toString() || new Date().toISOString(),
      isFinished: mediaProgress.isFinished || false,
    },
  };
}

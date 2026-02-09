import { HardcoverAudiobook } from './hardcoverTypes';
import { createAudiobookshelfClient } from './services/audiobookshelf/audiobookshelfClient';
import { createHardcoverClient } from './services/hardcover/hardcoverClient';

export interface AppConfig {
  audiobookshelf: {
    url: string;
    apiKey: string;
    userId: string;
  };
  hardcover: {
    apiUrl: string;
    apiKey: string;
  };
  syncInterval: string;
  logDir: string;
  apiTimeout: number;
}

export interface FormattedBook {
  book: {
    title: string;
    isbn: string | null;
    asin: string | null;
    authors: string[];
  };
  progress: {
    percent: number;
    currentTimeSeconds: number;
    durationSeconds: number;
    lastUpdate: string | Date;
    isFinished: boolean;
    startedAt?: string;
    finishedAt?: string;
  };
}

export interface HardcoverBook {
  id: string;
  title: string;
  slug: string;
  description?: string;
  pages?: number;
  release_date?: string;
  contributions?: Array<{
    author: {
      name: string;
    };
  }>;
  isbns?: string[];
  alternative_titles?: string[];
  image?: {
    url: string;
  };
}

export interface HardcoverGraphQLResponse<T> {
  data: T;
  errors?: Array<{
    message: string;
    locations?: Array<{
      line: number;
      column: number;
    }>;
    path?: string[];
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    extensions?: any;
  }>;
}

export interface SyncCache {
  audiobookDetails: Map<string, { formattedBook: FormattedBook }>;
  hardcoverAudiobooks: HardcoverAudiobook[] | null;
  hardcoverUserId: string | null;
}

export type AudiobookshelfClient = ReturnType<typeof createAudiobookshelfClient>;

export type HardcoverClient = ReturnType<typeof createHardcoverClient>;

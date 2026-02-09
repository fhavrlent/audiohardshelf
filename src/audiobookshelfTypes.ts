// Auto-generated TypeScript interfaces for AudioBookShelf API models

// Basic reusable types

/** File metadata for any library file */
export interface FileMetadata {
  filename: string;
  ext: string;
  path: string;
  relPath: string;
  size: number;
  mtimeMs: number;
  ctimeMs: number;
  birthtimeMs: number;
}

/** ID3 and similar metadata tags from audio files */
export interface AudioMetaTags {
  tagAlbum?: string | null;
  tagArtist?: string | null;
  tagGenre?: string | null;
  tagTitle?: string | null;
  tagSeries?: string | null;
  tagSeriesPart?: string | null;
  tagTrack?: string | null;
  tagDisc?: string | null;
  tagSubtitle?: string | null;
  tagAlbumArtist?: string | null;
  tagDate?: string | null;
  tagComposer?: string | null;
  tagPublisher?: string | null;
  tagComment?: string | null;
  tagDescription?: string | null;
  tagEncoder?: string | null;
  tagEncodedBy?: string | null;
  tagIsbn?: string | null;
  tagLanguage?: string | null;
  tagASIN?: string | null;
  tagOverdriveMediaMarker?: string | null;
  tagOriginalYear?: string | null;
  tagReleaseCountry?: string | null;
  tagReleaseType?: string | null;
  tagReleaseStatus?: string | null;
  tagISRC?: string | null;
  tagMusicBrainzTrackId?: string | null;
  tagMusicBrainzAlbumId?: string | null;
  tagMusicBrainzAlbumArtistId?: string | null;
  tagMusicBrainzArtistId?: string | null;
}

/** A generic library file (audio, image, etc.) */
export interface LibraryFile {
  ino: string;
  metadata: FileMetadata;
  addedAt: number;
  updatedAt: number;
  fileType: string;
}

/** An audio track within a media stream */
export interface AudioTrack {
  index: number;
  startOffset: number;
  duration: number;
  title: string;
  contentUrl: string;
  mimeType: string;
  metadata: FileMetadata | null;
}

/** A chapter within an audiobook or audio file */
export interface BookChapter {
  id: number;
  start: number;
  end: number;
  title: string;
}

/** Audio file details for an audiobook segment */
export interface AudioFile {
  index: number;
  ino: string;
  metadata: FileMetadata;
  addedAt: number;
  updatedAt: number;
  trackNumFromMeta: number | null;
  discNumFromMeta: number | null;
  trackNumFromFilename: number | null;
  discNumFromFilename: number | null;
  manuallyVerified: boolean;
  exclude: boolean;
  error: string | null;
  format: string;
  duration: number;
  bitRate: number;
  language: string | null;
  codec: string;
  timeBase: string;
  channels: number;
  channelLayout: string;
  chapters: BookChapter[];
  embeddedCoverArt: string | null;
  metaTags: AudioMetaTags;
  mimeType: string;
}

/** EBook file details */
export interface EBookFile {
  ino: string;
  metadata: FileMetadata;
  ebookFormat: string;
  addedAt: number;
  updatedAt: number;
}

/** Enclosure info for podcast episode download */
export interface PodcastEpisodeEnclosure {
  url: string;
  type: string;
  length: string;
}

// Media types and enums

export type MediaType = 'book' | 'podcast';
export type EpisodeType = string;
export type PlayMethod = 0 | 1 | 2 | 3;
export type UserType = 'root' | 'guest' | 'user' | 'admin';

// Book Models

/** Detailed metadata for a book */
export interface BookMetadata {
  title: string | null;
  subtitle: string | null;
  authors: AuthorMinified[];
  narrators: string[];
  series: SeriesSequence[];
  genres: string[];
  publishedYear: string | null;
  publishedDate: string | null;
  publisher: string | null;
  description: string | null;
  isbn: string | null;
  asin: string | null;
  language: string | null;
  explicit: boolean;
}

/** Minified book metadata */
export interface BookMetadataMinified {
  title: string;
  titleIgnorePrefix: string;
  subtitle: string | null;
  authorName: string;
  authorNameLF: string;
  narratorName: string;
  seriesName: string;
  genres: string[];
  publishedYear: string | null;
  publishedDate: string | null;
  publisher: string | null;
  description: string | null;
  isbn: string | null;
  asin: string | null;
  language: string | null;
  explicit: boolean;
}

/** Expanded book metadata includes naming helpers */
export interface BookMetadataExpanded extends BookMetadata {
  titleIgnorePrefix: string;
  authorName: string;
  authorNameLF: string;
  narratorName: string;
  seriesName: string;
}

/** Full Book details */
export interface Book {
  libraryItemId: string;
  metadata: BookMetadata;
  coverPath: string | null;
  tags: string[];
  audioFiles: AudioFile[];
  chapters: BookChapter[];
  ebookFile: EBookFile | null;
}

/** Minified book for listings */
export interface BookMinified {
  metadata: BookMetadataMinified;
  coverPath: string | null;
  tags: string[];
  numTracks: number;
  numAudioFiles: number;
  numChapters: number;
  duration: number;
  size: number;
  ebookFormat: string | null;
}

/** Expanded book with full details and track listing */
export interface BookExpanded {
  libraryItemId: string;
  metadata: BookMetadataExpanded;
  coverPath: string | null;
  tags: string[];
  audioFiles: AudioFile[];
  chapters: BookChapter[];
  duration: number;
  size: number;
  tracks: AudioTrack[];
  ebookFile: EBookFile | null;
}

// Podcast Models

/** Metadata for a podcast series */
export interface PodcastMetadata {
  title: string | null;
  author: string | null;
  description: string | null;
  releaseDate: string | null;
  genres: string[];
  feedUrl: string | null;
  imageUrl: string | null;
  itunesPageUrl: string | null;
  itunesId: number | null;
  itunesArtistId: number | null;
  explicit: boolean;
  language: string | null;
  type: string | null;
}

/** Minified podcast metadata includes prefix helper */
export interface PodcastMetadataMinified extends PodcastMetadata {
  titleIgnorePrefix: string;
}

/** Expanded podcast metadata adds prefix helper */
export interface PodcastMetadataExpanded extends PodcastMetadata {
  titleIgnorePrefix: string;
}

/** Podcast details */
export interface Podcast {
  libraryItemId: string;
  metadata: PodcastMetadata;
  coverPath: string | null;
  tags: string[];
  episodes: PodcastEpisode[];
  autoDownloadEpisodes: boolean;
  autoDownloadSchedule?: string;
  lastEpisodeCheck: number;
  maxEpisodesToKeep: number;
  maxNewEpisodesToDownload: number;
}

/** Minified podcast for listings */
export interface PodcastMinified {
  metadata: PodcastMetadataMinified;
  coverPath: string | null;
  tags: string[];
  numEpisodes: number;
  autoDownloadEpisodes: boolean;
  autoDownloadSchedule?: string;
  lastEpisodeCheck: number;
  maxEpisodesToKeep: number;
  maxNewEpisodesToDownload: number;
  size: number;
}

/** Expanded podcast with full episode details */
export interface PodcastExpanded {
  libraryItemId: string;
  metadata: PodcastMetadataExpanded;
  coverPath: string | null;
  tags: string[];
  episodes: PodcastEpisodeExpanded[];
  autoDownloadEpisodes: boolean;
  autoDownloadSchedule?: string;
  lastEpisodeCheck: number;
  maxEpisodesToKeep: number;
  maxNewEpisodesToDownload: number;
  size: number;
}

/** Podcast episode summary */
export interface PodcastEpisode {
  libraryItemId: string;
  id: string;
  index: number;
  season: string;
  episode: string;
  episodeType: EpisodeType;
  title: string;
  subtitle: string;
  description: string;
  enclosure: PodcastEpisodeEnclosure;
  pubDate: string;
  audioFile: AudioFile;
  publishedAt: number;
  addedAt: number;
  updatedAt: number;
}

/** Podcast episode with expanded media details */
export interface PodcastEpisodeExpanded extends PodcastEpisode {
  audioTrack: AudioTrack;
  duration: number;
  size: number;
}

/** Podcast episode download request */
export interface PodcastEpisodeDownload {
  id: string;
  episodeDisplayTitle: string;
  url: string;
  libraryItemId: string;
  libraryId: string;
  isFinished: boolean;
  failed: boolean;
  startedAt: number | null;
  createdAt: number;
  finishedAt: number | null;
  podcastTitle: string | null;
  podcastExplicit: boolean;
  season: string | null;
  episode: string | null;
  episodeType: EpisodeType;
  publishedAt: number | null;
}

/** Podcast feed from source */
export interface PodcastFeed {
  metadata: PodcastFeedMetadata;
  episodes: PodcastFeedEpisode[];
}

/** Minified podcast feed */
export interface PodcastFeedMinified {
  metadata: PodcastFeedMetadata;
  numEpisodes: number;
}

/** Metadata for podcast feed */
export interface PodcastFeedMetadata {
  image: string;
  categories: string[];
  feedUrl: string;
  description: string;
  descriptionPlain: string;
  title: string;
  language: string;
  explicit: string;
  author: string;
  pubDate: string;
  link: string;
}

/** Minified podcast feed metadata */
export type PodcastFeedMetadataMinified = Pick<
  PodcastFeedMetadata,
  'title' | 'description' | 'descriptionPlain'
> & {
  preventIndexing: boolean;
  ownerName: string | null;
  ownerEmail: string | null;
};

/** Episode entry in RSS feed */
export interface PodcastFeedEpisode {
  title: string;
  subtitle: string;
  description: string;
  descriptionPlain: string;
  pubDate: string;
  episodeType: EpisodeType;
  season: string;
  episode: string;
  author: string;
  duration: string;
  explicit: string;
  publishedAt: number;
  enclosure: PodcastEpisodeEnclosure;
}

// Library and organization

/** A user's library */
export interface Library {
  id: string;
  name: string;
  folders: Folder[];
  displayOrder: number;
  icon: string;
  mediaType: MediaType;
  provider: string;
  settings: LibrarySettings;
  createdAt: number;
  lastUpdate: number;
}

/** Settings for a library */
export interface LibrarySettings {
  coverAspectRatio: 0 | 1;
  disableWatcher: boolean;
  skipMatchingMediaWithAsin: boolean;
  skipMatchingMediaWithIsbn: boolean;
  autoScanCronExpression: string | null;
}

/** Filter data for library views */
export interface LibraryFilterData {
  authors: AuthorMinified[];
  genres: string[];
  tags: string[];
  series: SeriesSequence[];
  narrators: string[];
  languages: string[];
}

/** A watched folder in a library */
export interface Folder {
  id: string;
  fullPath: string;
  libraryId: string;
  addedAt: number;
}

// Library items

/** Full library item */
export interface LibraryItem {
  id: string;
  ino: string;
  libraryId: string;
  folderId: string;
  path: string;
  relPath: string;
  isFile: boolean;
  mtimeMs: number;
  ctimeMs: number;
  birthtimeMs: number;
  addedAt: number;
  updatedAt: number;
  lastScan: number | null;
  scanVersion: string | null;
  isMissing: boolean;
  isInvalid: boolean;
  mediaType: MediaType;
  media: Book | Podcast;
  libraryFiles: LibraryFile[];
}

/** Minified library item */
export interface LibraryItemMinified {
  id: string;
  ino: string;
  libraryId: string;
  folderId: string;
  path: string;
  relPath: string;
  isFile: boolean;
  mtimeMs: number;
  ctimeMs: number;
  birthtimeMs: number;
  addedAt: number;
  updatedAt: number;
  isMissing: boolean;
  isInvalid: boolean;
  mediaType: MediaType;
  media: BookMinified | PodcastMinified;
  numFiles: number;
  size: number;
}

/** Expanded library item with detailed media and size */
export interface LibraryItemExpanded {
  id: string;
  ino: string;
  libraryId: string;
  folderId: string;
  path: string;
  relPath: string;
  isFile: boolean;
  mtimeMs: number;
  ctimeMs: number;
  birthtimeMs: number;
  addedAt: number;
  updatedAt: number;
  lastScan: number | null;
  scanVersion: string | null;
  isMissing: boolean;
  isInvalid: boolean;
  mediaType: MediaType;
  media: BookExpanded | PodcastExpanded;
  libraryFiles: LibraryFile[];
  size: number;
}

// Author and series

/** Author details */
export interface Author {
  id: string;
  asin: string | null;
  name: string;
  description: string | null;
  imagePath: string | null;
  addedAt: number;
  updatedAt: number;
}

/** Minified author */
export interface AuthorMinified {
  id: string;
  name: string;
}

/** Expanded author with book count */
export interface AuthorExpanded extends Author {
  numBooks: number;
}

/** A series grouping of books */
export interface Series {
  id: string;
  name: string;
  description: string | null;
  addedAt: number;
  updatedAt: number;
}

/** Series with book IDs and count */
export interface SeriesNumBooks {
  id: string;
  name: string;
  nameIgnorePrefix: string;
  libraryItemIds: string[];
  numBooks: number;
}

/** Series with actual book items and total duration */
export interface SeriesBooks extends Series {
  nameIgnorePrefix: string;
  nameIgnorePrefixSort: string;
  type: 'series';
  books: LibraryItemExpanded[];
  totalDuration: number;
}

/** Association of a book to a series sequence position */
export interface SeriesSequence {
  id: string;
  name: string;
  sequence: string | null;
}

// Collections and playlists

/** A named collection of books */
export interface Collection {
  id: string;
  libraryId: string;
  userId: string;
  name: string;
  description: string | null;
  books: LibraryItemMinified[];
  lastUpdate: number;
  createdAt: number;
}

/** Expanded collection with full item details */
export interface CollectionExpanded extends Omit<Collection, 'books'> {
  books: LibraryItemExpanded[];
}

/** A playlist of media items/episodes */
export interface Playlist {
  id: string;
  libraryId: string;
  userId: string;
  name: string;
  description: string | null;
  coverPath: string | null;
  items: PlaylistItem[];
  lastUpdate: number;
  createdAt: number;
}

/** Expanded playlist with linked item details */
export interface PlaylistExpanded extends Omit<Playlist, 'items'> {
  items: PlaylistItemExpanded[];
}

/** Item reference in a playlist */
export interface PlaylistItem {
  libraryItemId: string;
  episodeId: string | null;
}

/** Expanded playlist item with nested details */
export interface PlaylistItemExpanded extends PlaylistItem {
  libraryItem: LibraryItemMinified | LibraryItemExpanded;
  episode?: PodcastEpisodeExpanded;
}

// User progress and playback

/** Track consumption progress of media */
export interface MediaProgress {
  id: string;
  userId: string;
  libraryItemId: string;
  episodeId: string | null;
  mediaItemId: string;
  mediaItemType: 'book' | 'podcast';
  duration: number;
  progress: number;
  currentTime: number;
  isFinished: boolean;
  hideFromContinueListening: boolean;
  ebookLocation: string | null;
  ebookProgress: number;
  lastUpdate: number;
  startedAt: number;
  finishedAt: number | null;
}

/** Media progress including full media details */
export interface MediaProgressWithMedia extends MediaProgress {
  media: BookExpanded | PodcastExpanded;
  episode?: PodcastEpisode;
}

/** Active playback session info */
export interface PlaybackSession {
  id: string;
  userId: string;
  libraryId: string;
  libraryItemId: string;
  episodeId: string | null;
  mediaType: MediaType;
  mediaMetadata: BookMetadata | PodcastMetadata;
  chapters: BookChapter[];
  displayTitle: string;
  displayAuthor: string;
  coverPath: string;
  duration: number;
  playMethod: PlayMethod;
  mediaPlayer: string;
  deviceInfo: DeviceInfo;
  serverVersion: string;
  date: string;
  dayOfWeek: string;
  timeListening: number;
  startTime: number;
  currentTime: number;
  startedAt: number;
  updatedAt: number;
}

/** Expanded playback session with tracks and item details */
export interface PlaybackSessionExpanded extends PlaybackSession {
  audioTracks: AudioTrack[];
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  videoTrack: any | null;
  libraryItem: LibraryItemExpanded;
}

/** Device information for playback session */
export interface DeviceInfo {
  id: string;
  userId: string;
  deviceId: string;
  ipAddress?: string | null;
  browserName?: string | null;
  browserVersion?: string | null;
  osName?: string | null;
  osVersion?: string | null;
  deviceName?: string | null;
  deviceType?: string | null;
  manufacturer?: string | null;
  model?: string | null;
  sdkVersion?: number | null;
  clientName: string;
  clientVersion: string;
}

/** User authentication and permissions */
export interface UserPermissions {
  download: boolean;
  update: boolean;
  delete: boolean;
  upload: boolean;
  accessAllLibraries: boolean;
  accessAllTags: boolean;
  accessExplicitContent: boolean;
}

/** Audio bookmark within a book */
export interface AudioBookmark {
  libraryItemId: string;
  title: string;
  time: number;
  createdAt: number;
}

/** A user of the system with full details */
export interface User {
  id: string;
  username: string;
  type: UserType;
  token: string;
  mediaProgress: MediaProgress[];
  seriesHideFromContinueListening: string[];
  bookmarks: AudioBookmark[];
  isActive: boolean;
  isLocked: boolean;
  lastSeen: number | null;
  createdAt: number;
  permissions: UserPermissions;
  librariesAccessible: string[];
  itemTagsAccessible: string[];
}

/** User with embedded media details */
export interface UserWithProgressDetails extends Omit<User, 'mediaProgress'> {
  mediaProgress: MediaProgressWithMedia[];
}

/** User public view with session */
export interface UserWithSession {
  id: string;
  username: string;
  type: UserType;
  session: PlaybackSessionExpanded | null;
  lastSeen: number | null;
  createdAt: number;
}

// Backup, notifications, server settings

/** Server backup summary */
export interface Backup {
  id: string;
  backupMetadataCovers: boolean;
  backupDirPath: string;
  datePretty: string;
  fullPath: string;
  path: string;
  filename: string;
  fileSize: number;
  createdAt: number;
  serverVersion: string;
}

/** Notification settings for Apprise */
export interface NotificationSettings {
  id: string;
  appriseType: string;
  appriseApiUrl: string | null;
  notifications: Notification[];
  maxFailedAttempts: number;
  maxNotificationQueue: number;
  notificationDelay: number;
}

/** Individual notification rule */
export interface Notification {
  id: string;
  libraryId: string | null;
  eventName: string;
  urls: string[];
  titleTemplate: string;
  bodyTemplate: string;
  enabled: boolean;
  type: string;
  lastFiredAt: number | null;
  lastAttemptFailed: boolean;
  numConsecutiveFailedAttempts: number;
  numTimesFired: number;
  createdAt: number;
}

/** Notification event definition */
export interface NotificationEvent {
  name: string;
  requiresLibrary: boolean;
  libraryMediaType?: MediaType;
  description: string;
  variables: string[];
  defaults: {
    title: string;
    body: string;
  };
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  testData: Record<string, any>;
}

/** Global server settings */
export interface ServerSettings {
  id: string;
  scannerFindCovers: boolean;
  scannerCoverProvider: string;
  scannerParseSubtitle: boolean;
  scannerPreferMatchedMetadata: boolean;
  scannerDisableWatcher: boolean;
  storeCoverWithItem: boolean;
  storeMetadataWithItem: boolean;
  metadataFileFormat: 'json' | 'abs';
  rateLimitLoginRequests: number;
  rateLimitLoginWindow: number;
  backupSchedule: string;
  backupsToKeep: number;
  maxBackupSize: number;
  loggerDailyLogsToKeep: number;
  loggerScannerLogsToKeep: number;
  homeBookshelfView: 0 | 1;
  bookshelfView: 0 | 1;
  sortingIgnorePrefix: boolean;
  sortingPrefixes: string[];
  chromecastEnabled: boolean;
  dateFormat: string;
  timeFormat: string;
  language: string;
  logLevel: number;
  version: string;
}

/** RSS Feed entry */
export interface RssFeed {
  id: string;
  slug: string;
  userId: string;
  entityType: string;
  entityId: string;
  coverPath: string;
  serverAddress: string;
  feedUrl: string;
  meta: RssFeedMetadata;
  episodes: RssFeedEpisode[];
  createdAt: number;
  updatedAt: number;
}

/** Minified RSS Feed listing */
export interface RssFeedMinified {
  id: string;
  entityType: string;
  entityId: string;
  feedUrl: string;
  meta: RssFeedMetadata;
}

/** Metadata for RSS feed */
export interface RssFeedMetadata {
  title: string;
  description: string;
  author: string | null;
  imageUrl: string;
  feedUrl: string;
  link: string;
  explicit: boolean;
  type: string | null;
  language: string | null;
  preventIndexing: boolean;
  ownerName: string | null;
  ownerEmail: string | null;
}

/** Minified RSS feed metadata */
export type RssFeedMetadataMinified = Pick<
  RssFeedMetadata,
  'title' | 'description' | 'preventIndexing' | 'ownerName' | 'ownerEmail'
>;

/** Episode in RSS feed */
export interface RssFeedEpisode {
  id: string;
  title: string;
  description: string;
  enclosure: PodcastEpisodeEnclosure;
  pubDate: string;
  link: string;
  author: string;
  explicit: boolean;
  duration: number;
  season: string | null;
  episode: string | null;
  episodeType: string | null;
  libraryItemId: string;
  episodeId: string | null;
  trackIndex: number;
  fullPath: string;
}

// Streaming

/** Active media stream details */
export interface Stream {
  id: string;
  userId: string;
  libraryItem: LibraryItemExpanded;
  episode: PodcastEpisodeExpanded | null;
  segmentLength: number;
  playlistPath: string;
  clientPlaylistUri: string;
  startTime: number;
  segmentStartNumber: number;
  isTranscodeComplete: boolean;
}

/** Progress of media stream transcoding */
export interface StreamProgress {
  stream: string;
  percent: string;
  chunks: string[];
  numSegments: number;
}

/** Response from /api/libraries endpoint */
export interface LibrariesResponse {
  libraries: Library[];
}

/** Response from /api/libraries/{id}/items with pagination */
export interface LibraryItemsResponse {
  results: LibraryItemMinified[];
  total: number;
  limit: number;
  page: number;
}

export interface ItemsInProgressResponse {
  libraryItems: InProgressLibraryItem[];
}

export type InProgressLibraryItem = LibraryItemMinified & {
  recentEpisode?: PodcastEpisode;
  progressLastUpdate: number;
};

export type AudiobookshelfItemResponse = LibraryItem | (LibraryItemExpanded & ExpandedExtras);

interface ExpandedExtras {
  userMediaProgress?: MediaProgress;
  rssFeed?: RssFeedMetadataMinified | null;
  episodesDownloading?: PodcastEpisodeDownload[];
  media: (BookExpandedWithAuthors | BookExpanded) | PodcastExpanded;
}

export interface BookExpandedWithAuthors extends Omit<BookExpanded, 'metadata'> {
  metadata: Omit<BookExpanded['metadata'], 'authors'> & {
    authors: Author[];
  };
}

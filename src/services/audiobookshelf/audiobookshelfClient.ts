import axios from 'axios';
import config from '../../config/config';
import logger from '../logger';

export function createAudiobookshelfClient() {
  const baseURL = config.audiobookshelf.url;
  const apiKey = config.audiobookshelf.apiKey;
  const userId = config.audiobookshelf.userId;

  if (!baseURL || !apiKey || !userId) {
    const missingConfigs: string[] = [];
    if (!baseURL) missingConfigs.push('ABS_URL');
    if (!apiKey) missingConfigs.push('ABS_API_KEY');
    if (!userId) missingConfigs.push('ABS_USER_ID');

    const errorMessage = `Missing Audiobookshelf configuration: ${missingConfigs.join(', ')}`;
    logger.error(errorMessage);
    throw new Error(errorMessage);
  }

  const formattedBaseURL = baseURL.endsWith('/') ? baseURL.slice(0, -1) : baseURL;

  const client = axios.create({
    baseURL: formattedBaseURL,
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    timeout: config.apiTimeout,
  });

  return {
    getClient: () => client,
    getBaseURL: () => formattedBaseURL,
    getUserId: () => userId,
  };
}

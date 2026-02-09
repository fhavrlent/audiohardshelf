import axios from 'axios';
import logger from '../logger';
import { extractErrorMessage } from '../../utils/errors';
import { AudiobookshelfClient } from '../../types';

export async function validateAudiobookshelfConnection(
  client: AudiobookshelfClient
): Promise<boolean> {
  try {
    logger.info('Validating connection to Audiobookshelf');
    const apiClient = client.getClient();

    const response = await apiClient.get<{ success: boolean }>('/ping');

    if (response.status === 200) {
      logger.info('Successfully connected to Audiobookshelf');
      return true;
    } else {
      logger.warn(`Unexpected response from Audiobookshelf: ${response.status}`);
      return false;
    }
  } catch (error) {
    const errorMessage = extractErrorMessage(error);
    const statusCode =
      axios.isAxiosError(error) && error.response ? error.response.status : 'unknown';
    const errorDetails = axios.isAxiosError(error) && error.response ? error.response.data : {};

    logger.error('Failed to connect to Audiobookshelf', {
      error: errorMessage,
      baseURL: client.getBaseURL(),
      statusCode,
      details: JSON.stringify(errorDetails),
    });

    if (statusCode === 404) {
      logger.error('Audiobookshelf API endpoint not found (404)', {
        currentUrl: client.getBaseURL(),
        suggestions: [
          'Check ABS_URL in .env file',
          'Verify Audiobookshelf server is running',
          'Ensure URL includes http:// or https://',
          'Check if server uses a different API path',
        ],
      });
    } else if (statusCode === 401 || statusCode === 403) {
      logger.error(`Authentication failed with Audiobookshelf (${statusCode})`, {
        suggestions: [
          'Check ABS_API_KEY in .env file',
          'Verify API key is valid and not expired',
          'Verify user has correct permissions',
        ],
      });
    }

    return false;
  }
}

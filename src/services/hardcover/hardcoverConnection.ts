import logger from '../logger';
import { extractErrorMessage } from '../../utils/errors';
import { Users } from '../../generated/graphql';
import { HardcoverClient } from '../../types';

export async function validateHardcoverConnection(client: HardcoverClient): Promise<boolean> {
  try {
    logger.info('Validating connection to Hardcover');

    const meQuery = `
      query {
        me {
          id
          username
          name
          books_count
        }
      }
    `;

    const result = await client.executeQuery<{
      me: Users[];
    }>(meQuery);

    if (result && result.me) {
      const user = Array.isArray(result.me) ? result.me[0] : result.me;

      if (user && user.id) {
        client.withUserId(user.id.toString());
        client.withUsername(user.username);

        logger.info(`Connected to Hardcover as ${user.username} (ID: ${user.id})`);
        logger.info(`User's book count: ${user.books_count}`);

        return true;
      }
    }

    if (result) {
      logger.warn(`Unexpected response structure: ${JSON.stringify(result)}`);
    } else {
      logger.warn('Received null or undefined response from Hardcover API');
    }
    return false;
  } catch (error) {
    const errorMessage = extractErrorMessage(error);

    logger.error('Failed to connect to Hardcover', {
      error: errorMessage,
      baseURL: client.getBaseURL(),
    });

    if (
      errorMessage.includes('401') ||
      errorMessage.includes('403') ||
      errorMessage.includes('authentication') ||
      errorMessage.includes('unauthorized')
    ) {
      logger.error('Authentication failed with Hardcover', {
        suggestions: [
          'Check HARDCOVER_API_KEY in .env file',
          'Verify API key is valid and not expired',
          'Verify user has correct permissions',
        ],
      });
    } else if (
      errorMessage.includes('ENOTFOUND') ||
      errorMessage.includes('ECONNREFUSED') ||
      errorMessage.includes('404')
    ) {
      logger.error('Could not reach Hardcover API', {
        suggestions: [
          'Check HARDCOVER_API_URL in .env file',
          'Verify internet connection',
          'Check Hardcover service status',
        ],
      });
    }

    return false;
  }
}

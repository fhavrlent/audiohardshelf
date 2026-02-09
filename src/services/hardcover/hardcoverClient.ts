import axios from 'axios';
import config from '../../config/config';
import logger from '../logger';
import { HardcoverGraphQLResponse } from '../../types';

export function createHardcoverClient() {
  let client: ReturnType<typeof initializeClient> | null = null;
  let userId: string | null = null;
  let username: string | null = null;

  function initializeClient() {
    const baseURL = config.hardcover.apiUrl;
    const apiKey = config.hardcover.apiKey;

    if (!baseURL) {
      const errorMessage = 'Missing Hardcover API URL in configuration';
      logger.error(errorMessage);
      throw new Error(errorMessage);
    }

    if (!apiKey) {
      const errorMessage = 'Missing Hardcover API key in configuration';
      logger.error(errorMessage);
      throw new Error(errorMessage);
    }

    const formattedApiKey = apiKey.startsWith('Bearer ') ? apiKey : `Bearer ${apiKey}`;

    logger.info(`Initialized Hardcover GraphQL client with base URL: ${baseURL}`);

    return axios.create({
      baseURL: baseURL,
      headers: {
        Authorization: formattedApiKey,
        'Content-Type': 'application/json',
        'User-Agent': 'AudioHardShelf/0.1 - Audiobook Sync Tool',
      },
      timeout: config.apiTimeout,
    });
  }

  function getClient() {
    if (!client) {
      client = initializeClient();
    }
    return client;
  }

  return {
    executeQuery: async <T>(query: string, variables?: Record<string, unknown>): Promise<T> => {
      try {
        const payload: Record<string, unknown> = { query };

        if (variables) {
          payload.variables = variables;
        }

        const response = await getClient().post<HardcoverGraphQLResponse<T>>('', payload);

        if (response.data.errors?.length) {
          const errorMessages = response.data.errors.map(e => e.message).join('; ');
          logger.error(`GraphQL errors: ${errorMessages}`, {
            errors: response.data.errors,
          });
          throw new Error(`GraphQL errors: ${errorMessages}`);
        }

        return response.data.data;
      } catch (error) {
        if (axios.isAxiosError(error)) {
          const statusCode = error.response?.status;
          const errorData = error.response?.data;

          logger.error(`GraphQL request failed`, {
            status: statusCode,
            error: error.message,
            data: JSON.stringify(errorData),
          });
        } else {
          logger.error(
            `GraphQL request error: ${error instanceof Error ? error.message : String(error)}`
          );
        }
        throw error;
      }
    },

    getBaseURL: () => config.hardcover.apiUrl,
    getUserId: () => userId,
    getUsername: () => username,

    withUserId: (newUserId: string) => {
      userId = newUserId;
      return userId;
    },

    withUsername: (newUsername: string) => {
      username = newUsername;
      return username;
    },
  };
}

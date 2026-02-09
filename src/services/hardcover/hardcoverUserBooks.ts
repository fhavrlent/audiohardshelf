import logger from '../logger';
import { logError } from '../../utils/errors';
import { HardcoverAudiobook } from '../../hardcoverTypes';
import { User_Books, Users } from '../../generated/graphql';
import { HardcoverClient } from '../../types';

async function getUserId(client: HardcoverClient): Promise<string> {
  try {
    const cachedUserId = client.getUserId();
    if (cachedUserId) {
      logger.debug(`Using cached Hardcover user ID: ${cachedUserId}`);
      return cachedUserId;
    }

    logger.info('Getting Hardcover user ID');

    const userIdQuery = `
      query {
        me {
          id
        }
      }
    `;

    const result = await client.executeQuery<{
      me: Users[];
    }>(userIdQuery);

    if (result && result.me) {
      const user = Array.isArray(result.me) ? result.me[0] : result.me;

      if (user && user.id) {
        const userId = user.id.toString();
        logger.info(`Found Hardcover user ID: ${userId}`);
        client.withUserId(userId);
        return userId;
      }
    }

    logger.warn('Failed to get user ID from Hardcover, response data is invalid');
    throw new Error('Invalid response data when getting user ID');
  } catch (error) {
    logError('Error getting Hardcover user ID', error);
    throw error;
  }
}

export async function getCurrentlyReadingBooks(
  client: HardcoverClient,
  userId?: string
): Promise<Array<User_Books>> {
  try {
    const userIdToUse = userId || (await getUserId(client));

    logger.info(`Fetching currently reading books for user ${userIdToUse}`);

    const currentlyReadingQuery = `
      query {
        user_books(where: {status_id: {_eq: 2}, user_id: {_eq: ${userIdToUse}}}) {
          edition_id
          book {
            title
            editions(where: {audio_seconds: {_gte: 0}}) {
              id
              isbn_10
              isbn_13
              asin
              book_id
              audio_seconds
              title
            }
            user_books(where: { user: {id: {_eq: ${userIdToUse}}}}) {
              id
              edition_id
            }
          }
        }
      }
    `;

    const result = await client.executeQuery<{ user_books: Array<User_Books> }>(
      currentlyReadingQuery
    );

    if (result && result.user_books && Array.isArray(result.user_books)) {
      logger.info(`Found ${result.user_books.length} books currently in progress on Hardcover`);
      return [...result.user_books];
    } else {
      logger.warn('Failed to get currently reading books, response data is invalid');
      return [];
    }
  } catch (error) {
    logError('Error fetching currently reading books', error, {
      userId,
    });
    return [];
  }
}

export async function getCurrentlyReadingAudiobooks(
  client: HardcoverClient,
  userId?: string
): Promise<HardcoverAudiobook[]> {
  try {
    const books = await getCurrentlyReadingBooks(client, userId);

    const audiobooks = books
      .filter(book => {
        if (!book.book.editions || book.book.editions.length === 0) {
          return false;
        }

        return book.book.editions.some(edition => edition.id === book.edition_id);
      })
      .map(book => {
        const matchingEdition = book.book.editions.find(edition => edition.id === book.edition_id);

        if (!matchingEdition) {
          logger.warn(
            `Expected to find matching audio edition for book "${book.book.title}" but none found`
          );
          return null;
        }

        const userBookId = book.book.user_books?.find(ub => ub.edition_id === book.edition_id)?.id;

        if (!userBookId) {
          logger.warn(
            `Expected to find user book ID for edition "${book.edition_id}" of book "${book.book.title}" but none found`
          );
          return null;
        }

        const otherUserEditionIds = book.book.user_books
          ? book.book.user_books.map(ub => ub.edition_id).filter(eid => eid !== book.edition_id)
          : [];

        return {
          edition_id: book.edition_id,
          user_book_id: userBookId,
          title: matchingEdition.title || book.book.title,
          isbn_10: matchingEdition.isbn_10,
          isbn_13: matchingEdition.isbn_13,
          asin: matchingEdition.asin,
          book_id: matchingEdition.book_id,
          audio_seconds: matchingEdition.audio_seconds,
          other_user_editions: otherUserEditionIds,
        };
      })
      .filter(Boolean) as HardcoverAudiobook[];

    const nonAudioCount = books.length - audiobooks.length;
    logger.info(
      `Found ${audiobooks.length} audiobooks currently in progress on Hardcover (${nonAudioCount} books are not audiobooks or are different editions)`
    );

    return audiobooks;
  } catch (error) {
    logError('Error processing currently reading audiobooks', error, {
      userId,
    });
    return [];
  }
}

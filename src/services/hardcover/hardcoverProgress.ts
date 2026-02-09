import { validateHardcoverConnection } from './hardcoverConnection';
import logger from '../logger';
import { logError } from '../../utils/errors';
import { User_Book_Reads } from '../../generated/graphql';
import { FormattedBook, HardcoverClient } from '../../types';

interface UpdateProgressInput {
  id: number;
  editionId: number;
  progressSeconds: number;
  startedAt?: string;
  finishedAt?: string;
}

export interface BookReadInfo {
  id: number;
  progressSeconds: number | null;
  startedAt?: string | null;
  finishedAt?: string | null;
}

export async function updateAudiobookProgress(
  client: HardcoverClient,
  input: UpdateProgressInput
): Promise<boolean> {
  try {
    logger.info(
      `Updating audiobook progress for edition ID: ${input.editionId}, progress: ${input.progressSeconds} seconds`
    );

    const mutation = `
      mutation UpdateUserBookReadMutation($id: Int!, $object: DatesReadInput!) {
        updateResult: update_user_book_read(id: $id, object: $object) {
          error
          userBookRead: user_book_read {
            id
            userBookId: user_book_id
            startedAt: started_at
            finishedAt: finished_at
            editionId: edition_id
            progressPages: progress_pages
            progressSeconds: progress_seconds
            __typename
          }
          __typename
        }
      }
    `;

    const variables = {
      id: input.id,
      object: {
        started_at: input.startedAt,
        finished_at: input.finishedAt,
        edition_id: input.editionId,
        progress_pages: 0,
        progress_seconds: input.progressSeconds,
      },
    };

    const result = await client.executeQuery<{
      updateResult: {
        error: string | null;
        userBookRead: {
          id: number;
          progressSeconds: number;
        } | null;
      };
    }>(mutation, variables);

    if (result.updateResult.error) {
      logger.error(`Error updating audiobook progress: ${result.updateResult.error}`);
      return false;
    }

    if (!result.updateResult.userBookRead) {
      logger.warn(`Updated audiobook progress but received no book data in response`);
      return true;
    }

    logger.info(
      `Successfully updated audiobook progress: ID ${result.updateResult.userBookRead.id}, ` +
        `Progress: ${result.updateResult.userBookRead.progressSeconds} seconds`
    );
    return true;
  } catch (error) {
    logError('Failed to update audiobook progress', error, {
      editionId: input.editionId,
      progressSeconds: input.progressSeconds,
    });
    return false;
  }
}

export async function getBookReadInfo({
  client,
  editionId,
  userId,
}: {
  client: HardcoverClient;
  editionId: number;
  userId?: string;
}): Promise<BookReadInfo | null> {
  try {
    logger.info(`Fetching book read info for edition ID: ${editionId}`);

    const userIdToUse = userId || client.getUserId();
    if (!userIdToUse) {
      logger.error('No user ID available to fetch book read info');
      return null;
    }

    const query = `
      query GetBookReadInfo($editionId: Int!, $userId: Int!) {
        user_book_reads(
          where: {
            edition_id: {_eq: $editionId},
            user_book: {user_id: {_eq: $userId}}
          }
        ) {
          id
          started_at
          finished_at
          progress_seconds
        }
      }
    `;

    const variables = {
      editionId,
      userId: parseInt(userIdToUse, 10),
    };

    const result = await client.executeQuery<{
      user_book_reads: Array<User_Book_Reads>;
    }>(query, variables);

    if (!result.user_book_reads?.length) {
      logger.warn(`No user_book_read found for edition ID: ${editionId}`);
      return null;
    }

    const bookRead = result.user_book_reads[0];
    const progressSeconds =
      bookRead.progress_seconds !== null && bookRead.progress_seconds !== undefined
        ? bookRead.progress_seconds
        : 0;

    logger.info(
      `Found book read info for edition ID: ${editionId} - ID: ${bookRead.id}, Progress: ${progressSeconds} seconds`
    );

    return {
      id: bookRead.id,
      progressSeconds: progressSeconds,
      startedAt: bookRead.started_at,
      finishedAt: bookRead.finished_at,
    };
  } catch (error) {
    logError('Failed to fetch book read info', error, {
      editionId,
    });
    return null;
  }
}

export async function updateAudiobookProgressByEditionId({
  client,
  editionId,
  progressSeconds,
  userId,
  bookReadInfo,
  formattedBook,
}: {
  client: HardcoverClient;
  editionId: number;
  progressSeconds: number;
  userId?: string;
  bookReadInfo: BookReadInfo;
  formattedBook: FormattedBook;
}): Promise<boolean> {
  try {
    let userIdToUse = userId || client.getUserId();

    if (!userIdToUse) {
      logger.debug('No user ID available, will verify Hardcover connection');
      const isValid = await validateHardcoverConnection(client);
      if (!isValid) {
        logger.error('Failed to validate Hardcover connection and get user ID');
        return false;
      }

      userIdToUse = client.getUserId();
      if (!userIdToUse) {
        logger.error('Still no user ID available after connection validation');
        return false;
      }

      logger.debug(`Using validated user ID: ${userIdToUse} for progress update`);
    } else {
      logger.debug(`Using provided user ID: ${userIdToUse} for progress update`);
    }

    const roundedNewProgress = Math.round(progressSeconds);

    const currentProgressSeconds =
      bookReadInfo.progressSeconds !== null ? bookReadInfo.progressSeconds : 0;
    if (currentProgressSeconds === roundedNewProgress) {
      logger.info(
        `Skipping update for edition ID: ${editionId} - progress already at ${roundedNewProgress} seconds`
      );
      return true;
    }

    return await updateAudiobookProgress(client, {
      id: bookReadInfo.id,
      editionId,
      progressSeconds: roundedNewProgress,
      startedAt: formattedBook.progress.startedAt ?? undefined,
      finishedAt: formattedBook.progress.finishedAt ?? undefined,
    });
  } catch (error) {
    logError('Failed to update audiobook progress by edition ID', error, {
      editionId,
      progressSeconds,
    });
    return false;
  }
}

export async function updateBookStatus({
  client,
  userBookId,
  editionId,
  statusId,
}: {
  client: HardcoverClient;
  userBookId: number;
  editionId: number;
  statusId: number;
}): Promise<boolean> {
  try {
    logger.info(
      `Updating book status for user book ID: ${userBookId}, edition ID: ${editionId}, new status: ${statusId}`
    );

    const mutation = `
      mutation UpdateUserBook($id: Int!, $object: UserBookUpdateInput!) {
        updateResponse: update_user_book(id: $id, object: $object) {
          error
          userBook: user_book {
            id
            bookId: book_id
            editionId: edition_id
            userId: user_id
            statusId: status_id
            __typename
          }
          __typename
        }
      }
    `;

    const variables = {
      id: userBookId,
      object: {
        edition_id: editionId,
        status_id: statusId,
      },
    };

    const result = await client.executeQuery<{
      updateResponse: {
        error: string | null;
        userBook: {
          id: number;
          statusId: number;
        } | null;
      };
    }>(mutation, variables);

    if (result.updateResponse.error) {
      logger.error(`Error updating book status: ${result.updateResponse.error}`);
      return false;
    }

    if (!result.updateResponse.userBook) {
      logger.warn(`Updated book status but received no book data in response`);
      return true;
    }

    logger.info(
      `Successfully updated book status: ID ${result.updateResponse.userBook.id}, ` +
        `New status: ${result.updateResponse.userBook.statusId}`
    );
    return true;
  } catch (error) {
    logError('Failed to update book status', error, {
      userBookId,
      editionId,
      statusId,
    });
    return false;
  }
}

import logger from '../logger';
import { logError } from '../../utils/errors';
import { HardcoverBook, HardcoverClient } from '../../types';
import { Maybe, SearchOutput } from '../../generated/graphql';

type SearchHit = { document: HardcoverBook };

export async function searchHardcoverBooks(
  client: HardcoverClient,
  query: string,
  queryType: string = 'Book',
  perPage: number = 20,
  page: number = 1
): Promise<HardcoverBook[]> {
  try {
    const searchQuery = `
      query Search($query: String!, $queryType: String!, $perPage: Int!, $page: Int!) {
        search(
          query: $query,
          query_type: $queryType,
          per_page: $perPage,
          page: $page
        ) {
          results
        }
      }
    `;

    const variables = {
      query,
      queryType,
      perPage,
      page,
    };

    const result = await client.executeQuery<{
      search: Maybe<SearchOutput>;
    }>(searchQuery, variables);

    if (result?.search?.results?.hits && Array.isArray(result.search.results.hits)) {
      const books = result.search.results.hits.map((hit: SearchHit) => hit.document);
      return [...books];
    } else {
      logger.debug(`No results found for "${query}"`);
      return [];
    }
  } catch (error) {
    logError(`Error searching Hardcover for "${query}"`, error, {
      query,
      queryType,
    });

    return [];
  }
}

export async function findBookInHardcover(
  client: HardcoverClient,
  book: Readonly<{
    title: string;
    isbn: string | null;
    asin: string | null;
    authors: string[];
  }>
): Promise<HardcoverBook | null> {
  const authorName = book.authors?.[0] ?? '';

  logger.info(`Looking for book in Hardcover: "${book.title}" by ${authorName}`);

  let searchResult: HardcoverBook | null = null;
  let searchStrategy = '';

  if (book.title) {
    logger.debug(`Trying search strategy: title only`);

    try {
      const titleResults = await searchHardcoverBooks(client, book.title);

      if (titleResults.length === 0) {
        return null;
      }

      if (titleResults.length === 1) {
        searchResult = titleResults[0];
        searchStrategy = 'title_exact_match';
        return searchResult;
      }

      if (book.isbn) {
        logger.debug(`Found ${titleResults.length} books, trying to narrow down by ISBN`);

        const isbnNormalized = book.isbn.replace(/-/g, '').trim();
        if (isbnNormalized.length > 5) {
          const isbnResults = await searchHardcoverBooks(client, isbnNormalized);

          if (isbnResults.length) {
            searchResult = isbnResults[0];
            searchStrategy = 'isbn_match';
            return searchResult;
          }
        } else {
          searchResult = titleResults[0];
          searchStrategy = 'title_first_match_invalid_isbn';
          return searchResult;
        }
      }

      searchResult = titleResults[0];
      searchStrategy = book.isbn ? 'title_first_match' : 'title_first_match_no_isbn';
      return searchResult;
    } catch (error) {
      logger.warn(`Error searching by title: ${book.title}`, {
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  if (!searchResult && book.title && authorName) {
    const combinedQuery = `${book.title} ${authorName}`;
    logger.debug(`Trying search strategy: title + author`);

    try {
      const results = await searchHardcoverBooks(client, combinedQuery);

      if (results.length) {
        searchResult = results[0];
        searchStrategy = 'title_author_match';
      }
    } catch (error) {
      logger.warn(`Error searching by title/author: ${combinedQuery}`, {
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  if (!searchResult && book.asin) {
    logger.debug(`Trying search strategy: ASIN`);

    try {
      const results = await searchHardcoverBooks(client, book.asin);

      if (results.length) {
        searchResult = results[0];
        searchStrategy = 'asin_match';
      }
    } catch (error) {
      logger.warn(`Error searching by ASIN: ${book.asin}`, {
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  }

  if (searchResult) {
    logger.info(
      `Found book in Hardcover: "${searchResult.title}" (ID: ${searchResult.id}, strategy: ${searchStrategy})`
    );
    return searchResult;
  } else {
    logger.warn(`Could not find book in Hardcover: "${book.title}" by ${authorName}`);
    return null;
  }
}

import dotenv from 'dotenv';
import path from 'path';
import fs from 'fs';
import { AppConfig } from '../types';

dotenv.config();

const readDockerSecret = (secretName: string): string | undefined => {
  const secretPath = `/run/secrets/${secretName}`;
  try {
    if (fs.existsSync(secretPath)) {
      return fs.readFileSync(secretPath, 'utf8').trim();
    }
    return undefined;
  } catch {
    return undefined;
  }
};

const config: AppConfig = {
  audiobookshelf: {
    url: process.env.ABS_URL || '',
    apiKey: readDockerSecret('abs_api_key') || process.env.ABS_API_KEY || '',
    userId: process.env.ABS_USER_ID || '',
  },
  hardcover: {
    apiUrl: 'https://api.hardcover.app/v1/graphql',
    apiKey: readDockerSecret('hardcover_api_key') || process.env.HARDCOVER_API_KEY || '',
  },
  syncInterval: process.env.SYNC_INTERVAL || '0 */1 * * *',
  logDir: path.join(__dirname, '../../logs'),
  apiTimeout: parseInt(process.env.API_TIMEOUT_MS || '10000', 10),
};

export default config;

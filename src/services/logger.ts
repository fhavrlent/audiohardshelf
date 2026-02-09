import winston from 'winston';
import DailyRotateFile from 'winston-daily-rotate-file';
import path from 'path';
import fs from 'fs';
import config from '../config/config';

const { format, transports } = winston;

if (!fs.existsSync(config.logDir)) {
  fs.mkdirSync(config.logDir, { recursive: true });
}

const logLevel = process.env.LOG_LEVEL || 'debug';

const maxLogFiles = process.env.LOG_MAX_FILES || '14d';
const maxLogSize = process.env.LOG_MAX_SIZE || '20m';

const logger = winston.createLogger({
  level: logLevel,
  format: format.combine(
    format.timestamp({ format: 'YYYY-MM-DD HH:mm:ss' }),
    format.errors({ stack: true }),
    format.splat(),
    format.json()
  ),
  defaultMeta: { service: 'audiohardshelf' },
  transports: [
    new transports.Console({
      format: format.combine(
        format.colorize(),
        format.printf(({ timestamp, level, message, ...rest }) => {
          return `${timestamp} ${level}: ${message} ${
            Object.keys(rest).length > 1 ? JSON.stringify(rest) : ''
          }`;
        })
      ),
    }),
    new DailyRotateFile({
      filename: path.join(config.logDir, 'error-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      level: 'error',
      maxSize: maxLogSize,
      maxFiles: maxLogFiles,
      zippedArchive: true,
    }),
    new DailyRotateFile({
      filename: path.join(config.logDir, 'combined-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      maxSize: maxLogSize,
      maxFiles: maxLogFiles,
      zippedArchive: true,
    }),
  ],
});

export default logger;

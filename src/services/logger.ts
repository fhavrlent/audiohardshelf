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

// Log retention configuration (configurable via env vars)
const maxLogFiles = process.env.LOG_MAX_FILES || '14d'; // Keep logs for 14 days by default
const maxLogSize = process.env.LOG_MAX_SIZE || '20m'; // Max 20MB per file by default

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
    // Rotating file transport for error logs
    new DailyRotateFile({
      filename: path.join(config.logDir, 'error-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      level: 'error',
      maxSize: maxLogSize,
      maxFiles: maxLogFiles,
      zippedArchive: true, // Compress old logs
    }),
    // Rotating file transport for all logs
    new DailyRotateFile({
      filename: path.join(config.logDir, 'combined-%DATE%.log'),
      datePattern: 'YYYY-MM-DD',
      maxSize: maxLogSize,
      maxFiles: maxLogFiles,
      zippedArchive: true, // Compress old logs
    }),
  ],
});

export default logger;

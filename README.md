# AudioHardShelf

A TypeScript application that automatically syncs your audiobook listening progress from Audiobookshelf to Hardcover.app.

## Features

- Syncs currently listening audiobooks from Audiobookshelf to Hardcover.app
- Matches books between platforms using ISBN, ASIN, or title search
- Updates progress percentages and timestamps
- Automatically marks books as "Finished"
- Runs on a configurable schedule (default: every hour)

## Prerequisites

- Node.js (LTS) - not needed if using Docker/Podman
- A self-hosted Audiobookshelf server with API access
- A Hardcover.app account with API access

## Installation & Usage

### Docker/Podman Installation (Recommended)

The recommended way to run AudioHardShelf is using Docker or Podman, which handles all dependencies and runs in an isolated environment.

1. Create or download existing docker-compose.yaml file
   ```yaml
   services:
      audiohardshelf:
         image: ghcr.io/fhavrlent/audiohardshelf:latest
         container_name: audiohardshelf
         restart: unless-stopped

         volumes:
            - logs-data:/app/logs:Z

         environment:
            - ABS_URL=${ABS_URL}
            - ABS_USER_ID=${ABS_USER_ID}
            - SYNC_INTERVAL=${SYNC_INTERVAL:-0 */1 * * *}
            - ABS_API_KEY=${ABS_API_KEY}
            - HARDCOVER_API_KEY=${HARDCOVER_API_KEY}
            # Optional:
            # - LOG_LEVEL=${LOG_LEVEL:-info}
            # - LOG_MAX_FILES=${LOG_MAX_FILES:-14d}
            # - LOG_MAX_SIZE=${LOG_MAX_SIZE:-20m}
            # - API_TIMEOUT_MS=${API_TIMEOUT_MS:-10000}

   volumes:
      logs-data:

   ```

2. Create a `.env` file by copying the example:
   ```bash
   cp .env.example .env
   ```
   
3. Edit the `.env` file with your own credentials:
   ```
   # Audiobookshelf configuration
   ABS_URL=http://your-audiobookshelf-server:port
   ABS_API_KEY=your_audiobookshelf_api_key
   ABS_USER_ID=your_audiobookshelf_user_id

   # Hardcover.app configuration
   HARDCOVER_API_KEY=your_hardcover_api_key
   
   # Sync configuration
   SYNC_INTERVAL="0 */1 * * *"  # Every hour by default
   ```

4. Run with Docker Compose:
   ```bash
   docker-compose up -d
   ```

   or with Podman Compose:
   ```bash
   podman-compose up -d
   ```

### PM2 Installation (Alternative)

If you prefer not to use containers, you can run AudioHardShelf using PM2, a process manager for Node.js applications:

1. Clone this repository to your VPS or local machine:
   ```bash
   git clone <repository-url> audiohardshelf
   cd audiohardshelf
   ```

2. Install dependencies:
   ```bash
   npm install
   ```

3. Create a `.env` file by copying the example:
   ```bash
   cp .env.example .env
   ```

4. Edit the `.env` file with your own credentials:
   ```
   # Audiobookshelf configuration
   ABS_URL=http://your-audiobookshelf-server:port
   ABS_API_KEY=your_audiobookshelf_api_key
   ABS_USER_ID=your_audiobookshelf_user_id

   # Hardcover.app configuration
   HARDCOVER_API_KEY=your_hardcover_api_key
   
   # Sync configuration
   SYNC_INTERVAL="0 */1 * * *"  # Every hour by default
   ```

5. Build the application:
   ```bash
   npm run build
   ```

6. Install PM2 (if not already installed):
   ```bash
   npm install -g pm2
   ```

7. Start your application with PM2:
   ```bash
   pm2 start dist/index.js --name audiohardshelf
   ```

8. Make it restart automatically on server reboot:
   ```bash
   pm2 startup
   pm2 save
   ```


## Configuration

You can adjust the following settings in your `.env` file:

### Required Settings
- `ABS_URL`: Your Audiobookshelf server URL
- `ABS_API_KEY`: Your Audiobookshelf API key (found in Settings > Users > [Your User] > API Token)
- `ABS_USER_ID`: Your Audiobookshelf user ID (found in the URL when viewing your profile)
- `HARDCOVER_API_KEY`: Your Hardcover API key (found in your account settings)

### Optional Settings
- `SYNC_INTERVAL`: Two formats supported (default: `0 */1 * * *`):
  - Cron pattern (e.g. `0 */1 * * *`): Runs at specific clock times (every hour on the hour)
  - Number of minutes (e.g. `60`): Runs every X minutes from when the service starts

- `API_TIMEOUT_MS`: API request timeout in milliseconds (default: `10000` = 10 seconds)
  - Increase if you have slow network or the servers are slow to respond

- `LOG_LEVEL`: Logging verbosity (default: `debug`)
  - Options: `error`, `warn`, `info`, `debug`
  - Use `error` or `warn` for production to reduce log volume

- `LOG_MAX_FILES`: How long to keep log files (default: `14d`)
  - Examples: `7d` (7 days), `30d` (30 days), `60d` (60 days)
  - Older logs are automatically deleted

- `LOG_MAX_SIZE`: Maximum size per log file before rotation (default: `20m`)
  - Examples: `10m` (10 megabytes), `50m` (50 megabytes)
  - Files exceeding this size are rotated immediately

## Logs

Logs are stored in the `logs` directory with automatic rotation and cleanup:

- **Log Files:**
  - `combined-YYYY-MM-DD.log`: All logs for a specific date
  - `error-YYYY-MM-DD.log`: Only error logs for a specific date
  - Older log files are compressed (`.gz`) to save space

- **Automatic Cleanup:**
  - Logs older than the retention period (default: 14 days) are automatically deleted
  - Maximum file size is enforced (default: 20MB per file)
  - This prevents unlimited log growth and saves disk space

- **Docker/Podman:**
  - The logs directory is mounted as a volume to persist logs outside the container
  - Compressed logs take ~70-90% less space

- **Customization:**
  - Use `LOG_MAX_FILES` to change retention period
  - Use `LOG_MAX_SIZE` to change file size limits
  - Use `LOG_LEVEL` to control verbosity (less verbose = smaller logs)

## How It Works

1. The application connects to your Audiobookshelf server and retrieves:
   - Books currently in progress
   - Their listening progress details (duration, percentage, etc.)

2. For each book, it attempts to find a match in your Hardcover.app library using:
   - ISBN (highest priority, if available)
   - ASIN (if available)
   - Exact title match (as fallback)

3. When a match is found, it synchronizes your current listening position with Hardcover.app

4. The synchronization runs automatically according to your configured schedule (default: hourly)
   - You can also trigger a manual sync using `npm run sync`

5. Detailed logs are kept in the `logs` directory

## Development

This project is written in TypeScript. To contribute to the development:

1. Make your changes in the `src` directory
2. Build the project:
   ```bash
   npm run build
   ```
3. Test your changes:
   ```bash
   npm run sync
   ```

## Troubleshooting

If books aren't syncing correctly:

1. Check the logs in the `logs` directory
2. Verify your API credentials are correct
3. Ensure your Audiobookshelf books have proper metadata (ISBN, ASIN, or accurate title/author)
4. Make sure the books exist in your Hardcover.app library

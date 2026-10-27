# AudioHardShelf

Keep your [Hardcover](https://hardcover.app/) reading progress in step with what you listen to on your self-hosted [Audiobookshelf](https://www.audiobookshelf.org/) server.

Listen in any Audiobookshelf app, and a minute after you pause, your progress on Hardcover catches up. Finish a book and it's marked as Read.

## Features

- **Near real-time:** listens to Audiobookshelf's live updates, so Hardcover is updated within minutes, not hours
- **Automatic matching** of your Audiobookshelf books to Hardcover by ISBN, ASIN or title
- **Finished books** are marked as Read on Hardcover, with start and finish dates
- **Re-read aware:** won't mark a book Read again if you've started re-reading it on Hardcover
- **Lightweight:** a single ~9 MB container image for amd64 and arm64

## Before You Start: How Books Are Matched

AudioHardShelf **updates** books on Hardcover. It doesn't add them. For a book to sync, all of the following must be true on Hardcover:

1. The book is on your shelf with the status **Currently Reading**.
2. You picked the **audiobook edition** you're listening to. Hardcover tracks audiobook progress in time, so print and ebook editions are skipped. You can change the edition from the book's page on Hardcover.
3. That edition can be matched to your Audiobookshelf book by one of the following, tried in this order:
   - **ISBN**: the ISBN in Audiobookshelf equals the edition's ISBN-10 or ISBN-13
   - **ASIN**: the ASIN in Audiobookshelf equals the edition's ASIN (Audible editions usually have one)
   - **Title**: the titles are exactly the same, ignoring capitalization

Tip: matching your books in Audiobookshelf against Audible (*Match* on a book, provider *Audible*) fills in the ASIN, which is the most reliable way to match audiobooks.

## Getting Started

### 1. Get your API keys

- **Audiobookshelf:** in the web UI, go to **Settings → API Keys** and create a key for your user (Audiobookshelf 2.26 or newer). Progress is synced for the user the key belongs to.
- **Hardcover:** sign in and open [hardcover.app/account/api](https://hardcover.app/account/api).

### 2. Create `docker-compose.yaml`

```yaml
services:
  audiohardshelf:
    image: ghcr.io/fhavrlent/audiohardshelf:latest
    container_name: audiohardshelf
    restart: unless-stopped

    environment:
      - ABS_URL=${ABS_URL}
      - SYNC_INTERVAL=${SYNC_INTERVAL:-1h}
      - ABS_API_KEY=${ABS_API_KEY}
      - HARDCOVER_API_KEY=${HARDCOVER_API_KEY}
      # Optional:
      # - LOG_LEVEL=${LOG_LEVEL:-debug}
      # - API_TIMEOUT_MS=${API_TIMEOUT_MS:-10000}
```

### 3. Create `.env` next to it

Docker Compose reads this file to fill in the variables above (see [`.env.example`](.env.example)):

```
ABS_URL=http://your-audiobookshelf-server:13378
ABS_API_KEY=your_audiobookshelf_api_key
HARDCOVER_API_KEY=your_hardcover_api_key
```

`ABS_URL` is the address you open Audiobookshelf at, including any sub-path (for example `https://example.com/audiobookshelf`). If Audiobookshelf runs in Docker on the same machine, you can also put both containers on the same Docker network and use the container's name, such as `http://audiobookshelf:80`.

### 4. Start it

```bash
docker compose up -d
docker logs -f audiohardshelf
```

On startup it runs a full sync and logs what it found. You should then see `listening for progress updates from Audiobookshelf`.

Podman works the same way with `podman-compose up -d`.

To pin a release instead of following `latest`, use a version tag such as `ghcr.io/fhavrlent/audiohardshelf:0.1.0`, or `:0.1` for patch updates only.

### Sync on demand

```bash
docker compose run --rm audiohardshelf sync
```

## Configuration

| Variable | Required | Default | Description |
|---|---|---|---|
| `ABS_URL` | yes | | Your Audiobookshelf server URL |
| `ABS_API_KEY` | yes | | Audiobookshelf API key |
| `HARDCOVER_API_KEY` | yes | | Hardcover API key; the `Bearer ` prefix is optional |
| `SYNC_INTERVAL` | | `1h` | How often to run a full sync regardless of activity, as a duration such as `30m` or `2h` (minimum `1m`). This is a fallback; see [When syncs run](#when-syncs-run). |
| `LOG_LEVEL` | | `debug` | `error`, `warn`, `info` or `debug` |
| `API_TIMEOUT_MS` | | `10000` | Timeout for each API request, in milliseconds |
| `HARDCOVER_API_URL` | | `https://api.hardcover.app/v1/graphql` | Hardcover GraphQL endpoint |

The API keys can also be provided as Docker secrets named `abs_api_key` and `hardcover_api_key`.

## How It Works

Each sync:

1. Fetches the books you're listening to in Audiobookshelf, plus books you finished in the last 30 days.
2. Fetches the audiobooks you're currently reading on Hardcover.
3. Matches them up (see [How books are matched](#before-you-start-how-books-are-matched)).
4. For each match, writes your listening position and start date to Hardcover. For finished books, it also sets the finish date and changes the status to **Read**.

Once a book is marked Read, it's no longer "Currently Reading" on Hardcover, so later syncs leave it alone.

### When syncs run

AudioHardShelf stays connected to Audiobookshelf's live updates, the same connection its web and mobile apps use:

- **While you listen**, it syncs once your progress has been unchanged for a minute (for example, when you pause), and at least every 10 minutes during long sessions.
- **When you mark a book finished** or change progress by hand, it syncs a minute later.
- **On startup, after a reconnect, and every `SYNC_INTERVAL`**, it runs a full sync to catch anything it missed.

If the live connection fails, the app keeps retrying in the background and still syncs every `SYNC_INTERVAL`.

## Limitations

- **It never adds books or starts them on Hardcover.** You need to mark a book as Currently Reading, in its audiobook edition, before it syncs.
- **Audiobook editions only.** Progress isn't synced to print or ebook editions on Hardcover, and Audiobookshelf ebook reading progress isn't synced at all.
- **Sync is one-way, Audiobookshelf → Hardcover.** Changes made on Hardcover never reach Audiobookshelf. Audiobookshelf always wins: if Hardcover shows further progress than Audiobookshelf, it's overwritten.
- **Progress is synced as a listening position in seconds.** Hardcover calculates the percentage from its edition's length, so if your file is a different length (another narrator, abridged, extra material), the percentage on Hardcover will be off.
- **Title matching is exact.** "Dune" won't match "Dune: Book One". Use ISBN or ASIN for anything with a subtitle or series in the title.
- **Finished books are picked up for 30 days.** If the app isn't running for longer than that, books finished in the meantime aren't marked Read; mark them on Hardcover yourself.
- **Re-reads:** if you finish a book and start the same day on a re-read in Hardcover, the new read may be marked Read again.
- **One user per container.** It syncs the Audiobookshelf user that owns the API key to one Hardcover account. For more people, run one container each.
- **Podcasts are ignored.**

## Troubleshooting

Set `LOG_LEVEL=debug` and check `docker logs audiohardshelf`; every book is logged with what happened to it.

| Log message | What to check |
|---|---|
| `no matching audiobook currently reading on Hardcover` | The book isn't Currently Reading on Hardcover, isn't using an audiobook edition, or no ISBN/ASIN/title matches. See [How books are matched](#before-you-start-how-books-are-matched). |
| `book has no read on Hardcover` | The book has no reading record on Hardcover to update. Give it a start date on Hardcover, or remove it and mark it Currently Reading again. |
| `check that ABS_API_KEY is valid` / `authentication failed, check ABS_API_KEY` | `ABS_API_KEY` is wrong, expired or disabled. |
| `check ABS_URL` / `connection refused` | `ABS_URL` is wrong, the server is down, or the URL is missing a sub-path. |
| `check that HARDCOVER_API_KEY is valid` / `connecting to Hardcover` | `HARDCOVER_API_KEY` is wrong or has expired; get a new one from your Hardcover account. |
| `Audiobookshelf socket disconnected` repeating | If Audiobookshelf is behind a reverse proxy, allow WebSocket connections to `/socket.io/`. Syncing still works every `SYNC_INTERVAL` without them. |

## Development

```bash
go test ./...
docker build -t audiohardshelf .
```

Pushing a version tag (e.g. `v0.1.0`) runs the tests and publishes the image to `ghcr.io/fhavrlent/audiohardshelf` as `0.1.0`, `0.1` and `latest`.

## License

[MIT](LICENSE)

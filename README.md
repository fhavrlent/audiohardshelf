# AudioHardShelf

Keep your [Hardcover](https://hardcover.app/) reading progress in step with what you listen to on your self-hosted [Audiobookshelf](https://www.audiobookshelf.org/) server.

Listen in any Audiobookshelf app, and a minute after you pause, your progress on Hardcover catches up. Finish a book and it's marked as Read.

## Features

- **Near real-time:** listens to Audiobookshelf's live updates, so Hardcover is updated within minutes, not hours
- **Automatic matching** of your Audiobookshelf books to Hardcover by ISBN, ASIN, or title and author
- **Optional auto-start:** books you start listening to are added to Hardcover as Currently Reading, in their audiobook edition
- **Finished books** are marked as Read on Hardcover, with start and finish dates
- **Correct percentages** even when your file is longer or shorter than the Hardcover edition
- **Re-read aware:** won't mark a book Read again if you've started re-reading it on Hardcover
- **Lightweight:** a single ~9 MB container image for amd64 and arm64

## Before You Start: How Books Are Matched

By default AudioHardShelf **updates** books on Hardcover. It doesn't add them. For a book to sync, all of the following must be true on Hardcover:

1. The book is on your shelf with the status **Currently Reading**.
2. You picked the **audiobook edition** you're listening to (reading format *Listened*). Hardcover tracks audiobook progress in time, so print and ebook editions are skipped. You can change the edition from the book's page on Hardcover.
3. That edition can be matched to your Audiobookshelf book by one of the following, tried in this order:
   - **ISBN**: the ISBN in Audiobookshelf equals the edition's ISBN-10 or ISBN-13
   - **ASIN**: the ASIN in Audiobookshelf equals the edition's ASIN (Audible editions usually have one)
   - **Title**: the titles are the same, ignoring capitalization, and the books don't list different authors
   - **Title and author**: the titles are the same once punctuation, a leading "The", "A" or "An", and bracketed parts like "(Book 1)" are ignored, or one is the other without its subtitle ("Dune" and "Dune: Book One"). At least one author must be the same in both.

Tip: matching your books in Audiobookshelf against Audible (*Match* on a book, provider *Audible*) fills in the ASIN, which is the most reliable way to match audiobooks.

### Auto-start

Set `AUTO_START=true` to skip steps 1 and 2. When you listen to a book that isn't Currently Reading on Hardcover, AudioHardShelf looks for its audiobook edition on Hardcover:

1. by the book's **ISBN or ASIN**, if it has one;
2. otherwise by searching Hardcover for its **title and first author**. A result counts only if it shares an author and its title, an alternative title or one of its audiobook editions' titles matches (see *Title and author* above), so books without ISBN or ASIN, such as most Czech audiobooks, and translations whose Hardcover entry uses the original title still work. When the book has several audiobook editions, the one whose title matches is preferred.

Then:

- **Not on your shelf:** adds it as Currently Reading in that edition.
- **Want to Read:** changes it to Currently Reading in that edition.
- **Currently Reading in another edition** (print or ebook): switches it to the audiobook edition.
- **Read, Paused, Did Not Finish or Ignored:** leaves it alone. Change the status on Hardcover if you want it to sync.

Only books you've listened to in the last 14 days are started, so books you abandoned in Audiobookshelf don't show up on Hardcover. A book that can't be started is tried again after a day. Books that Hardcover has no audiobook edition for can't be started: add the edition on Hardcover (reading format *Listened*), and the book is picked up on the next try.

With auto-start on, a Currently Reading book that has no reading record on Hardcover gets one.

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
      # - AUTO_START=${AUTO_START:-false}
      # - LOG_LEVEL=${LOG_LEVEL:-debug}
      # - API_TIMEOUT_MS=${API_TIMEOUT_MS:-10000}

    volumes:
      - audiohardshelf-data:/data

volumes:
  audiohardshelf-data:
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
| `AUTO_START` | | `false` | `true` to put books you listen to on Hardcover as Currently Reading. See [Auto-start](#auto-start). |
| `SYNC_INTERVAL` | | `1h` | How often to run a full sync regardless of activity, as a duration such as `30m` or `2h` (minimum `1m`). This is a fallback; see [When syncs run](#when-syncs-run). |
| `LOG_LEVEL` | | `debug` | `error`, `warn`, `info` or `debug` |
| `API_TIMEOUT_MS` | | `10000` | Timeout for each API request, in milliseconds |
| `HARDCOVER_API_URL` | | `https://api.hardcover.app/v1/graphql` | Hardcover GraphQL endpoint |
| `STATE_FILE` | | `/data/state.json` | Where sync state is kept between restarts. See [Sync state](#sync-state). |

The API keys can also be provided as Docker secrets named `abs_api_key` and `hardcover_api_key`.

## How It Works

Each sync:

1. Fetches the books you're listening to in Audiobookshelf, plus books you finished in the last 30 days (or since the last sync, if that was longer ago).
2. Fetches the audiobooks you're currently reading on Hardcover.
3. Matches them up (see [How books are matched](#before-you-start-how-books-are-matched)), starting unmatched books on Hardcover if [auto-start](#auto-start) is on.
4. For each match whose position changed since the last sync, writes your listening position and start date to Hardcover. For finished books, it also sets the finish date and changes the status to **Read**.

Once a book is marked Read, it's no longer "Currently Reading" on Hardcover, so later syncs leave it alone.

### Progress and edition length

Hardcover shows progress as a percentage of its edition's length. When your Audiobookshelf file is more than 5% longer or shorter than that (another narrator, abridged, extra material), your position is scaled to the edition's length so the percentage matches. Finished books are set to the edition's full length.

### Sync state

AudioHardShelf remembers what it last synced for each book in `STATE_FILE`, so it only contacts Hardcover when your progress actually changed, marks each finished listen as Read only once, and catches up on books finished while it wasn't running.

The compose file above keeps it in a named volume. If `/data` isn't mounted, the image's anonymous volume is used, which is lost when the container is recreated. If the directory doesn't exist at all (for example when running the binary outside Docker), state is kept in memory only. Deleting the file is safe; the next sync rebuilds it.

### When syncs run

AudioHardShelf stays connected to Audiobookshelf's live updates, the same connection its web and mobile apps use:

- **While you listen**, it syncs once your progress has been unchanged for a minute (for example, when you pause), and at least every 10 minutes during long sessions.
- **When you mark a book finished** or change progress by hand, it syncs a minute later.
- **On startup, after a reconnect, and every `SYNC_INTERVAL`**, it runs a full sync to catch anything it missed.

If the live connection fails, the app keeps retrying in the background and still syncs every `SYNC_INTERVAL`.

## Limitations

- **Audiobook editions only.** Progress isn't synced to print or ebook editions on Hardcover, and Audiobookshelf ebook reading progress isn't synced at all.
- **Without auto-start, it never adds books or starts them on Hardcover.** With it, only books that have an audiobook edition on Hardcover can be started, and a book without ISBN or ASIN needs at least one author in Audiobookshelf.
- **Audiobook editions without a length** on Hardcover still sync, but Hardcover can't show a percentage for them, and progress isn't scaled.
- **Sync is one-way, Audiobookshelf → Hardcover.** Changes made on Hardcover never reach Audiobookshelf. When your Audiobookshelf position changes, it overwrites the one on Hardcover, even if Hardcover's was further along. Progress you change on Hardcover alone is kept until you listen again.
- **Scaled progress is approximate.** When the lengths differ, the position on Hardcover is right as a percentage, not as a timestamp.
- **Loose title matching needs authors.** Titles that differ by more than punctuation, articles, a series marker or a subtitle won't match. Use ISBN or ASIN for those.
- **Finished books are picked up for 30 days without a state file.** With [sync state](#sync-state) kept, books finished while the app wasn't running are caught up on however long it was.
- **One user per container.** It syncs the Audiobookshelf user that owns the API key to one Hardcover account. For more people, run one container each.
- **Podcasts are ignored.**

## Troubleshooting

Set `LOG_LEVEL=debug` and check `docker logs audiohardshelf`; every book is logged with what happened to it.

| Log message | What to check |
|---|---|
| `no matching audiobook currently reading on Hardcover` | The book isn't Currently Reading on Hardcover, isn't using an audiobook edition, or no ISBN/ASIN/title matches. See [How books are matched](#before-you-start-how-books-are-matched). |
| `book has no read on Hardcover` | The book has no reading record on Hardcover to update. Give it a start date on Hardcover, remove it and mark it Currently Reading again, or turn on `AUTO_START`. |
| `not starting book on Hardcover: the book is there but has no audiobook edition` | Hardcover only has print or ebook editions. Add an audiobook edition (reading format *Listened*) on Hardcover. |
| `not starting book on Hardcover: not found by ISBN, ASIN, or title and author` | Check the title and author in Audiobookshelf against Hardcover, or add the book on Hardcover yourself. |
| `not starting book on Hardcover: no audiobook edition with its ISBN or ASIN, and no author to search by` | Add the author in Audiobookshelf. |
| `not starting book on Hardcover: it has another status there` | Auto-start doesn't touch books that are Read, Paused, Did Not Finish or Ignored. Set it to Currently Reading or Want to Read on Hardcover. |
| `saving state failed` | `STATE_FILE`'s directory isn't writable. Use a named volume as in the compose file above, not a bind mount owned by root. |
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

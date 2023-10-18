# tstor (WIP)

tstor is an advanced remote torrent clien for self-hosting enthusiasts.

It expose virtual filesystem with torrents and archives presented as fully featured directories with limited amount of mutability. Virtual filesystem can be exported as a webDAV, HTTP endpoint or NFS(WIP).

tstor is based on amazing [distribyted](https://github.com/distribyted/distribyted), but has more focus on store a torrent data when streaming it.

## Special thanks

- [distribyted](https://github.com/distribyted/distribyted)
- [Anacrolix BitTorrent client package and utilities](https://github.com/anacrolix/torrent-repo-url). An amazing torrent library with file seek support.
- [Nwaples rardecode library, experimental branch](https://github.com/nwaples/rardecode/tree/experimental). The only go library that is able to seek over rar files and avoid to use `io.Discard`.
- [Bodgit 7zip library](https://github.com/bodgit/sevenzip). Amazing library to decode 7zip files.

## License

Distributed under the GPL3 license. See `LICENSE` for more information.

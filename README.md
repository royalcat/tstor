[![Releases][releases-shield]][releases-url]
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL3 License][license-shield]][license-url]
[![Coveralls][coveralls-shield]][coveralls-url]
[![Docker Image][docker-pulls-shield]][docker-pulls-url]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://git.kmsign.ru/royalcat/tstor">
    <img src="mkdocs/docs/images/tstor_icon.png" alt="Logo" width="100">
  </a>

  <h3 align="center">tstor</h3>

  <p align="center">
    Torrent client with on-demand file downloading as a filesystem.
    <br />
    <br />
    <a href="https://git.kmsign.ru/royalcat/tstor/issues">Report a Bug</a>
    ·
    <a href="https://git.kmsign.ru/royalcat/tstor/issues">Request Feature</a>
  </p>
</p>

## About The Project

![tstor Screen Shot][product-screenshot]

tstor is an alternative torrent client.
It can expose torrent files as a standard FUSE, webDAV or HTTP endpoint and download them on demand, allowing random reads using a fixed amount of disk space.

tstor tries to make easier integrations with other applications using torrent files, presenting them as a standard filesystem.

**Note that tstor is in beta version, it is a proof of concept with a lot of bugs.**

## Use Cases

- Play **multimedia files** on your favorite video or audio player. These files will be downloaded on demand and only the needed parts.
- Explore TBs of data from public **datasets** only downloading the parts you need. Use **Jupyter Notebooks** directly to process or analyze this data.
- Give access to your latest dataset creation just by sharing a magnet link. People will start using your data in seconds.
- Play your **ROM backups** directly from the torrent file. You can have virtually GBs in games and only downloaded the needed ones.

## Documentation

Check [here][main-url] for further documentation.

## Contributing

Contributions are what make the open-source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

Some areas need more care than others:

- Windows and macOS tests and compatibility. I don't have any easy way to test tstor on these operating systems.
- Web interface. Web development is not my _forte_.
- Tutorials. Share with the community your use case!

## Special thanks

- [Anacrolix BitTorrent client package and utilities][torrent-repo-url]. An amazing torrent library with file seek support.
- [Nwaples rardecode library, experimental branch][rardecode-repo-url]. The only go library that is able to seek over rar files and avoid to use `io.Discard`.
- [Bodgit 7zip library][sevenzip-repo-url]. Amazing library to decode 7zip files.

## License

Distributed under the GPL3 license. See `LICENSE` for more information.

[sevenzip-repo-url]: https://github.com/bodgit/sevenzip
[rardecode-repo-url]: https://github.com/nwaples/rardecode/tree/experimental
[torrent-repo-url]: https://github.com/anacrolix/torrent
[main-url]: https://tstor.com
[releases-shield]: https://img.shields.io/github/v/release/tstor/tstor.svg?style=flat-square
[releases-url]: https://git.kmsign.ru/royalcat/tstor/releases
[docker-pulls-shield]: https://img.shields.io/docker/pulls/tstor/tstor.svg?style=flat-square
[docker-pulls-url]: https://hub.docker.com/r/tstor/tstor
[contributors-shield]: https://img.shields.io/github/contributors/tstor/tstor.svg?style=flat-square
[contributors-url]: https://git.kmsign.ru/royalcat/tstor/graphs/contributors
[forks-shield]: https://img.shields.io/github/forks/tstor/tstor.svg?style=flat-square
[forks-url]: https://git.kmsign.ru/royalcat/tstor/network/members
[stars-shield]: https://img.shields.io/github/stars/tstor/tstor.svg?style=flat-square
[stars-url]: https://git.kmsign.ru/royalcat/tstor/stargazers
[issues-shield]: https://img.shields.io/github/issues/tstor/tstor.svg?style=flat-square
[issues-url]: https://git.kmsign.ru/royalcat/tstor/issues
[releases-url]: https://git.kmsign.ru/royalcat/tstor/releases
[license-shield]: https://img.shields.io/github/license/tstor/tstor.svg?style=flat-square
[license-url]: https://git.kmsign.ru/royalcat/tstor/blob/master/LICENSE
[product-screenshot]: mkdocs/docs/images/tstor.gif
[example-config]: https://git.kmsign.ru/royalcat/tstor/blob/master/examples/conf_example.yaml
[coveralls-shield]: https://img.shields.io/coveralls/github/tstor/tstor?style=flat-square
[coveralls-url]: https://coveralls.io/github/tstor/tstor

# NKN Node Sampler

[![GitHub license](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/nknorg/nkn-node-sampler)](https://goreportcard.com/report/github.com/nknorg/nkn-node-sampler) [![Build Status](https://github.com/nknorg/nkn-node-sampler/actions/workflows/build-ubuntu.yml/badge.svg)](https://github.com/nknorg/nkn-node-sampler/actions/workflows/build-ubuntu.yml) [![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](#contributing)

![nkn](logo.png)

NKN node crawler with random sampling. Get an estimate of the following network
stats within just a few seconds and minimal resource (CPU, RAM, network):

- Total number of nodes in the network
- Total number of messages being relayed by the network per second

## Build

```shell
make
```

## Usage

```shell
./nkn-node-sampler
```

### Control concurrency and sampling size

```shell
./nkn-node-sampler -m 8 -n 8
```

- `m`: how many concurrency network request to make
- `n`: how many steps to sample
- `m x n` controls the sampling size

## Contributing

**Can I submit a bug, suggestion or feature request?**

Yes. Please open an issue for that.

**Can I contribute patches?**

Yes, we appreciate your help! To make contributions, please fork the repo, push
your changes to the forked repo with signed-off commits, and open a pull request
here.

Please sign off your commit. This means adding a line "Signed-off-by: Name
<email>" at the end of each commit, indicating that you wrote the code and have
the right to pass it on as an open source patch. This can be done automatically
by adding -s when committing:

```shell
git commit -s
```

## Community

- [Forum](https://forum.nkn.org/)
- [Discord](https://discord.gg/c7mTynX)
- [Telegram](https://t.me/nknorg)
- [Reddit](https://www.reddit.com/r/nknblockchain/)
- [Twitter](https://twitter.com/NKN_ORG)

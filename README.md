## Go EthereumAI

Official golang implementation of the EthereumAI protocol.

[![API Reference](
https://camo.githubusercontent.com/915b7be44ada53c290eb157634330494ebe3e30a/68747470733a2f2f676f646f632e6f72672f6769746875622e636f6d2f676f6c616e672f6764646f3f7374617475732e737667
)](https://godoc.org/github.com/ethereumai/go-ethereumai)
[![Go Report Card](https://goreportcard.com/badge/github.com/ethereumai/go-ethereumai)](https://goreportcard.com/report/github.com/ethereumai/go-ethereumai)
[![Travis](https://travis-ci.org/ethereumai/go-ethereumai.svg?branch=master)](https://travis-ci.org/ethereumai/go-ethereumai)
[![Gitter](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/ethereumai/go-ethereumai?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge)

Automated builds are available for stable releases and the unstable master branch.

## Building the source

For prerequisites and detailed build instructions please read the
[Installation Instructions](https://github.com/ethereumai/go-ethereumai/wiki/Building-EthereumAI)
on the wiki.

Building geai requires both a Go (version 1.7 or later) and a C compiler.
You can install them using your favourite package manager.
Once the dependencies are installed, run

    make geai

or, to build the full suite of utilities:

    make all

## Executables

The go-ethereumai project comes with several wrappers/executables found in the `cmd` directory.

| Command    | Description |
|:----------:|-------------|
| **`geai`** | Our main EthereumAI CLI client. It is the entry point into the EthereumAI network (main-, test- or private net), capable of running as a full node (default) archive node (retaining all historical state) or a light node (retrieving data live). It can be used by other processes as a gateway into the EthereumAI network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `geai --help` and the [CLI Wiki page](https://github.com/ethereumai/go-ethereumai/wiki/Command-Line-Options) for command line options. |
| `abigen` | Source code generator to convert EthereumAI contract definitions into easy to use, compile-time type-safe Go packages. It operates on plain [EthereumAI contract ABIs](https://github.com/ethereumai/wiki/wiki/EthereumAI-Contract-ABI) with expanded functionality if the contract bytecode is also available. However it also accepts Solidity source files, making development much more streamlined. Please see our [Native DApps](https://github.com/ethereumai/go-ethereumai/wiki/Native-DApps:-Go-bindings-to-EthereumAI-contracts) wiki page for details. |
| `bootnode` | Stripped down version of our EthereumAI client implementation that only takes part in the network node discovery protocol, but does not run any of the higher level application protocols. It can be used as a lightweight bootstrap node to aid in finding peers in private networks. |
| `evm` | Developer utility version of the EVM (EthereumAI Virtual Machine) that is capable of running bytecode snippets within a configurable environment and execution mode. Its purpose is to allow isolated, fine-grained debugging of EVM opcodes (e.g. `evm --code 60ff60ff --debug`). |
| `rlpdump` | Developer utility tool to convert binary RLP ([Recursive Length Prefix](https://github.com/ethereumai/wiki/wiki/RLP)) dumps (data encoding used by the EthereumAI protocol both network as well as consensus wise) to user friendlier hierarchical representation (e.g. `rlpdump --hex CE0183FFFFFFC4C304050583616263`). |
| `swarm`    | swarm daemon and tools. This is the entrypoint for the swarm network. `swarm --help` for command line options and subcommands. See https://swarm-guide.readthedocs.io for swarm documentation. |
| `puppeai`    | a CLI wizard that aids in creating a new EthereumAI network. |

## Running geai

Going through all the possible command line flags is out of scope here (please consult our
[CLI Wiki page](https://github.com/ethereumai/go-ethereumai/wiki/Command-Line-Options)), but we've
enumerated a few common parameter combos to get you up to speed quickly on how you can run your
own Geai instance.

### Full node on the main EthereumAI network

By far the most common scenario is people wanting to simply interact with the EthereumAI network:
create accounts; transfer funds; deploy and interact with contracts. For this particular use-case
the user doesn't care about years-old historical data, so we can fast-sync quickly to the current
state of the network. To do so:

```
$ geai console
```

This command will:

 * Start geai in fast sync mode (default, can be changed with the `--syncmode` flag), causing it to
   download more data in exchange for avoiding processing the entire history of the EthereumAI network,
   which is very CPU intensive.
 * Start up Geai's built-in interactive [JavaScript console](https://github.com/ethereumai/go-ethereumai/wiki/JavaScript-Console),
   (via the trailing `console` subcommand) through which you can invoke all official [`web3` methods](https://github.com/ethereumai/wiki/wiki/JavaScript-API)
   as well as Geai's own [management APIs](https://github.com/ethereumai/go-ethereumai/wiki/Management-APIs).
   This too is optional and if you leave it out you can always attach to an already running Geai instance
   with `geai attach`.

### Configuration

As an alternative to passing the numerous flags to the `geai` binary, you can also pass a configuration file via:

```
$ geai --config /path/to/your_config.toml
```

To get an idea how the file should look like you can use the `dumpconfig` subcommand to export your existing configuration:

```
$ geai --your-favourite-flags dumpconfig
```

*Note: This works only with geai v1.6.0 and above.*

#### Docker quick start

One of the quickest ways to get EthereumAI up and running on your machine is by using Docker:

```
docker run -d --name ethereumai-node -v /Users/alice/ethereumai:/root \
           -p 9545:9545 -p 40404:40404 \
           ethereumai/client-go
```

This will start geai in fast-sync mode with a DB memory allowance of 1GB just as the above command does.  It will also create a persistent volume in your home directory for saving your blockchain as well as map the default ports. There is also an `alpine` tag available for a slim version of the image.

Do not forget `--rpcaddr 0.0.0.0`, if you want to access RPC from other containers and/or hosts. By default, `geai` binds to the local interface and RPC endpoints is not accessible from the outside.

### Programatically interfacing Geai nodes

As a developer, sooner rather than later you'll want to start interacting with Geai and the EthereumAI
network via your own programs and not manually through the console. To aid this, Geai has built-in
support for a JSON-RPC based APIs ([standard APIs](https://github.com/ethereumai/wiki/wiki/JSON-RPC) and
[Geai specific APIs](https://github.com/ethereumai/go-ethereumai/wiki/Management-APIs)). These can be
exposed via HTTP, WebSockets and IPC (unix sockets on unix based platforms, and named pipes on Windows).

The IPC interface is enabled by default and exposes all the APIs supported by Geai, whereas the HTTP
and WS interfaces need to manually be enabled and only expose a subset of APIs due to security reasons.
These can be turned on/off and configured as you'd expect.

HTTP based JSON-RPC API options:

  * `--rpc` Enable the HTTP-RPC server
  * `--rpcaddr` HTTP-RPC server listening interface (default: "localhost")
  * `--rpcport` HTTP-RPC server listening port (default: 9545)
  * `--rpcapi` API's offered over the HTTP-RPC interface (default: "eai,net,web3")
  * `--rpccorsdomain` Comma separated list of domains from which to accept cross origin requests (browser enforced)
  * `--ws` Enable the WS-RPC server
  * `--wsaddr` WS-RPC server listening interface (default: "localhost")
  * `--wsport` WS-RPC server listening port (default: 8546)
  * `--wsapi` API's offered over the WS-RPC interface (default: "eai,net,web3")
  * `--wsorigins` Origins from which to accept websockets requests
  * `--ipcdisable` Disable the IPC-RPC server
  * `--ipcapi` API's offered over the IPC-RPC interface (default: "admin,debug,eai,miner,net,personal,shh,txpool,web3")
  * `--ipcpath` Filename for IPC socket/pipe within the datadir (explicit paths escape it)

You'll need to use your own programming environments' capabilities (libraries, tools, etc) to connect
via HTTP, WS or IPC to a Geai node configured with the above flags and you'll need to speak [JSON-RPC](http://www.jsonrpc.org/specification)
on all transports. You can reuse the same connection for multiple requests!

**Note: Please understand the security implications of opening up an HTTP/WS based transport before
doing so! Hackers on the internet are actively trying to subvert EthereumAI nodes with exposed APIs!
Further, all browser tabs can access locally running webservers, so malicious webpages could try to
subvert locally available APIs!**

### Operating a private network

Maintaining your own private network is more involved as a lot of configurations taken for granted in
the official networks need to be manually set up.

#### Defining the private genesis state

First, you'll need to create the genesis state of your networks, which all nodes need to be aware of
and agree upon. This consists of a small JSON file (e.g. call it `genesis.json`):

```json
{
  "config": {
        "chainId": 0,
        "homesteadBlock": 0,
        "eip155Block": 0,
        "eip158Block": 0
    },
  "alloc"      : {},
  "coinbase"   : "0x0000000000000000000000000000000000000000",
  "difficulty" : "0x20000",
  "extraData"  : "",
  "gasLimit"   : "0x2fefd8",
  "nonce"      : "0x0000000000000042",
  "mixhash"    : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "parentHash" : "0x0000000000000000000000000000000000000000000000000000000000000000",
  "timestamp"  : "0x00"
}
```

The above fields should be fine for most purposes, although we'd recommend changing the `nonce` to
some random value so you prevent unknown remote nodes from being able to connect to you. If you'd
like to pre-fund some accounts for easier testing, you can populate the `alloc` field with account
configs:

```json
"alloc": {
  "0x0000000000000000000000000000000000000001": {"balance": "111111111"},
  "0x0000000000000000000000000000000000000002": {"balance": "222222222"}
}
```

With the genesis state defined in the above JSON file, you'll need to initialize **every** Geai node
with it prior to starting it up to ensure all blockchain parameters are correctly set:

```
$ geai init path/to/genesis.json
```

#### Creating the rendezvous point

With all nodes that you want to run initialized to the desired genesis state, you'll need to start a
bootstrap node that others can use to find each other in your network and/or over the internet. The
clean way is to configure and run a dedicated bootnode:

```
$ bootnode --genkey=boot.key
$ bootnode --nodekey=boot.key
```

With the bootnode online, it will display an [`enode` URL](https://github.com/ethereumai/wiki/wiki/enode-url-format)
that other nodes can use to connect to it and exchange peer information. Make sure to replace the
displayed IP address information (most probably `[::]`) with your externally accessible IP to get the
actual `enode` URL.

*Note: You could also use a full fledged Geai node as a bootnode, but it's the less recommended way.*

#### Starting up your member nodes

With the bootnode operational and externally reachable (you can try `telnet <ip> <port>` to ensure
it's indeed reachable), start every subsequent Geai node pointed to the bootnode for peer discovery
via the `--bootnodes` flag. It will probably also be desirable to keep the data directory of your
private network separated, so do also specify a custom `--datadir` flag.

```
$ geai --datadir=path/to/custom/data/folder --bootnodes=<bootnode-enode-url-from-above>
```

*Note: Since your network will be completely cut off from the main and test networks, you'll also
need to configure a miner to process transactions and create new blocks for you.*

#### Running a private miner

Mining on the public EthereumAI network is a complex task as it's only feasible using GPUs, requiring
an OpenCL or CUDA enabled `eaiminer` instance. For information on such a setup, please consult the
[EtherAIMining subreddit](https://www.reddit.com/r/EtherAIMining/) and the [Genoil miner](https://github.com/Genoil/cpp-ethereumai)
repository.

In a private network setting however, a single CPU miner instance is more than enough for practical
purposes as it can produce a stable stream of blocks at the correct intervals without needing heavy
resources (consider running on a single thread, no need for multiple ones either). To start a Geai
instance for mining, run it with all your usual flags, extended by:

```
$ geai <usual-flags> --mine --minerthreads=1 --etheraibase=0x0000000000000000000000000000000000000000
```

Which will start mining blocks and transactions on a single CPU thread, crediting all proceedings to
the account specified by `--etheraibase`. You can further tune the mining by changing the default gas
limit blocks converge to (`--targetgaslimit`) and the price transactions are accepted at (`--gasprice`).

## Contribution

Thank you for considering to help out with the source code! We welcome contributions from
anyone on the internet, and are grateful for even the smallest of fixes!

If you'd like to contribute to go-ethereumai, please fork, fix, commit and send a pull request
for the maintainers to review and merge into the main code base. If you wish to submit more
complex changes though, please check up with the core devs first on [our gitter channel](https://gitter.im/ethereumai/go-ethereumai)
to ensure those changes are in line with the general philosophy of the project and/or get some
early feedback which can make both your efforts much lighter as well as our review and merge
procedures quick and simple.

Please make sure your contributions adhere to our coding guidelines:

 * Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
 * Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
 * Pull requests need to be based on and opened against the `master` branch.
 * Commit messages should be prefixed with the package(s) they modify.
   * E.g. "eai, rpc: make trace configs optional"

Please see the [Developers' Guide](https://github.com/ethereumai/go-ethereumai/wiki/Developers'-Guide)
for more details on configuring your environment, managing project dependencies and testing procedures.

## License

The go-ethereumai library (i.e. all code outside of the `cmd` directory) is licensed under the
[GNU Lesser General Public License v3.0](https://www.gnu.org/licenses/lgpl-3.0.en.html), also
included in our repository in the `COPYING.LESSER` file.

The go-ethereumai binaries (i.e. all code inside of the `cmd` directory) is licensed under the
[GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0.en.html), also included
in our repository in the `COPYING` file.

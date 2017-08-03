prefixout
=========

Prefixes stuff with stuff.

What?
-----

* Asciinema: https://asciinema.org/a/131927

Install
-------

```sh
go get -u -x -v github.com/akatrevorjay/prefixout
```

Usage
-----

```sh
$ prefixout --help
Usage: prefixout [-dtc] [-p PREFIX | --prefix PREFIX] -- COMMAND [ARGS ...]
        Prefixes stdout/stderr of a command. Nuff said.
Arguments:
        PREFIX          Prefix (defaults to "COMMAND: ")
        COMMAND         Command to exec
        ARGS            Command arguments
        -d                      Differentiate stdout/stderr
        -t                      Timestamp output
        -c                      Color output
```


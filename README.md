go-acb
======
acb extract library. port code from [VGMToolBox](https://sourceforge.net/projects/vgmtoolbox/) to Go

Installation
------------

    go get github.com/vazrupe/go-acb

Usage
-----
Import Library: 

    import (
        ...
        "github.com/vazrupe/go-acb/acb"
        ...
    )
    ...
    f, err := acb.LoadCriAcbFile(_your file path_)
    if err != nil {
        _load error_
    }
    ...

Commandline Use:

    go-acb [-f] [-save=_your save dir_] _acb files..._

and examples dir

Lisence
-------
MIT Lisence.

Reference
---------
VGMToolBox, https://sourceforge.net/projects/vgmtoolbox/
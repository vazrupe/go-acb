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
    f, err := acb.LoadCriAcbFile(YOUR_FILE_PATH)
    if err != nil {
        _load error_
    }
    ...

Commandline Use:

    go-acb [-f] [-save=YOUR_SAVE_DIR] ACB_FILEs...

and examples dir

Lisence
-------
MIT Lisence.

Reference
---------
VGMToolBox, https://sourceforge.net/projects/vgmtoolbox/

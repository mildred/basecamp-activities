Basecamp Activities Project Management
======================================

This manages the Activities project, update TODO items, ask for updates and
generates a weekly recap.

Hacking
=======

The dependencies are vendored. To vendor dependencies, just run:

    make deps

Docker Container
================

This program is designed to be run in a Docker container. To generate a Docker
image, run with sufficient permissions:

    make docker-image

If you want to create a debian package, you'll need to configure authnetication
settings. Just copy `config.mk.sample` to `config.mk` and update the variables
(Makefile syntax).

To create the package, run:

    make docker-package

To deploy it on a server, make sure the server details are set in `config.mk`
and run:

    make deploy

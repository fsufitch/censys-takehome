`censys-takehome`
=================

Implementation Notes
--------------------

I restructured the forked project a little bit, to illustrate my understanding of best practices.

### Less focus on Docker, more on general container tooling

I replaced `docker-compose.yml` and `Dockerfile` with `compose.yml` and `Containerfile`. The semantics are the same, but it emphasizes that the intent is for this to work with any general container tech, not just with Docker.

**Why?** These days, Docker is increasingly becoming a more closed/controlled ecosystem. Additionally, the Docker Engine continues to have a number of downsides (such as running as `root`, trouble with UID/GID mapping in volumes) which make for an awkward or insecure experience. I personally prefer Podman when possible.

### Moved `Containerfile` to the project root

To me, the `Containerfile` is like a `Makefile`; it belongs where it can "refer" to the entire project, i.e. at the root.

### Upgraded to Go 1.22

Go provides support going back two minor versions. Keeping up to date on the version being used is a good idea.

# Development and Release Process

## Development Process

The Kudo Project is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release
2. All [OWNERS](OWNERS) must LGTM this release
3. An OWNER runs `git tag -s $VERSION` and inserts the changelog and pushes the tag with `git push $VERSION`
4. The corresponding docker image `kudobuilder/controller:$VERSION` is pushed to dockerhub
5. The release issue is closed
6. An announcement email is sent to `kudobuilder@googlegroups.com` with the subject `[ANNOUNCE] Kudo $VERSION is released`

## Release Process

The official binaries for Kudo are created using [goreleaser](https://goreleaser.com/) for the release process through the circleci release job. The [.goreleaser.yml](.goreleaser.yml) defines the binaries which are supported for each release.

It is possible outside of the standard release process to build a "snapshot" release using the following command: `goreleaser release --skip-publish --snapshot --rm-dist`
This process will create a "dist" folder with all the build artifacts. The changelog is not created unless a full release is executed. If you are looking to get a "similar" changelog, install [github-release-notes](https://github.com/buchanae/github-release-notes) and execute `github-release-notes -org kudobuilder -repo kudo -since-latest-release`.

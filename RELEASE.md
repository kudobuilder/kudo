# Release Process

The Kudo Project is released on an as-needed basis. The process is as follows:

1. An issue is proposing a new release with a changelog since the last release
2. All [OWNERS](OWNERS) must LGTM this release
3. An OWNER runs `git tag -s $VERSION` and inserts the changelog and pushes the tag with `git push $VERSION`
4. The corresponding docker image `kudobuilder/controller:$VERSION` is pushed to dockerhub
5. The release issue is closed
6. An announcement email is sent to `kudobuilder@googlegroups.com` with the subject `[ANNOUNCE] Kudo $VERSION is released`

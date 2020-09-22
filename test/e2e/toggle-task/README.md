This test needs to be in the e2e section for now, as the integration test kuttl only starts a etcd and an api-server.

As the DeleteTask that is used by this test is doing a ForegroundDelete which requires the garbage collector, it would
fail as an integration test.
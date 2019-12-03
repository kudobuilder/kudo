resource "src-git": {
  type: "git"
  param url: "$(context.git.url)"
  param revision: "$(context.git.commit)"
}

task "test": {
  inputs: ["src-git"]
  steps: [
    {
      name: "test"
      image: "kudobuilder/golang:1.13"
      command: [ "make", "test" ],
      workingDir: "/workspace/src-git"
    }
  ]
}

task "lint": {
  inputs: ["src-git"]
  steps: [
    {
      name: "lint"
      image: "kudobuilder/golang:1.13"
      command: [ "make", "lint" ],
      workingDir: "/workspace/src-git"
    }
  ]
}

task "integration-test": {
  inputs: ["src-git"]
  steps: [
    {
      name: "test"
      image: "kudobuilder/golang:1.13"
      command: [ "./test/run_tests.sh", "integration-test" ],
      env: [
        {
          name: "INTEGRATION_OUTPUT_JUNIT"
          value: "true"
        }
      ]
      workingDir: "/workspace/src-git"
    }
  ]
}

task "e2e-test": {
  inputs: ["src-git"]
  steps: [
    {
      name: "test"
      image: "kudobuilder/golang:1.13"
      command: [ "./test/run_tests.sh", "e2e-test" ],
      env: [
        {
          name: "INTEGRATION_OUTPUT_JUNIT"
          value: "true"
        }
      ]
      workingDir: "/workspace/src-git"
    }
  ]
}

actions: [
  {
    tasks: ["test", "lint", "integration-test", "e2e-test"]
    on push branches: ["master"]
  },
  {
    tasks: ["test", "lint", "integration-test", "e2e-test"]
    on pull_request chatops: ["test"]
  }
]
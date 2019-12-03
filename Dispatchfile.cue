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
      image: "golangci/golangci-lint:v1.21.0"
      command: [ "golangci-lint", "run", "-v" ],
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
      command: [ "make", "integration-test" ],
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
    tasks: ["test", "lint", "integration-test"]
    on push branches: ["master"]
  },
  {
    tasks: ["test", "lint", "integration-test"]
    on pull_request chatops: ["test"]
  }
]
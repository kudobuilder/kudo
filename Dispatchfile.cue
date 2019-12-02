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

actions: [
  {
    tasks: ["test"]
    on push branches: ["master"]
  },
  {
    tasks: ["test"]
    on pull_request chatops: ["test"]
  }
]
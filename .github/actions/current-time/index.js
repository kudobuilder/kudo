const core = require("@actions/core");
const github = require("@actions/github");

try {
  const time = new Date().toISOString();
  core.setOutput("time", time);
} catch (error) {
  core.setFailed(error.message);
}

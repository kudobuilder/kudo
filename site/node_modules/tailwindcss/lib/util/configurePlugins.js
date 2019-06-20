"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;

function _default(pluginConfig, plugins) {
  const pluginNames = Array.isArray(pluginConfig) ? pluginConfig : Object.keys(plugins).filter(pluginName => {
    return pluginConfig !== false && pluginConfig[pluginName] !== false;
  });
  return pluginNames.map(pluginName => plugins[pluginName]());
}
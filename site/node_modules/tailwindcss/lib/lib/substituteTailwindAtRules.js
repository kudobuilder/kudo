"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;

var _lodash = _interopRequireDefault(require("lodash"));

var _postcss = _interopRequireDefault(require("postcss"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function updateSource(nodes, source) {
  return _lodash.default.tap(Array.isArray(nodes) ? _postcss.default.root({
    nodes
  }) : nodes, tree => {
    tree.walk(node => node.source = source);
  });
}

function _default(config, {
  base: pluginBase,
  components: pluginComponents,
  utilities: pluginUtilities
}) {
  return function (css) {
    css.walkAtRules('tailwind', atRule => {
      if (atRule.params === 'preflight') {
        // prettier-ignore
        throw atRule.error("`@tailwind preflight` is not a valid at-rule in Tailwind v1.0, use `@tailwind base` instead.", {
          word: 'preflight'
        });
      }

      if (atRule.params === 'base') {
        atRule.before(updateSource(pluginBase, atRule.source));
        atRule.remove();
      }

      if (atRule.params === 'components') {
        atRule.before(updateSource(pluginComponents, atRule.source));
        atRule.remove();
      }

      if (atRule.params === 'utilities') {
        atRule.before(updateSource(pluginUtilities, atRule.source));
        atRule.remove();
      }
    });
  };
}
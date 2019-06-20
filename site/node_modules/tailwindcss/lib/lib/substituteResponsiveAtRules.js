"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;

var _lodash = _interopRequireDefault(require("lodash"));

var _postcss = _interopRequireDefault(require("postcss"));

var _cloneNodes = _interopRequireDefault(require("../util/cloneNodes"));

var _buildMediaQuery = _interopRequireDefault(require("../util/buildMediaQuery"));

var _buildSelectorVariant = _interopRequireDefault(require("../util/buildSelectorVariant"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _default(config) {
  return function (css) {
    const {
      theme: {
        screens
      },
      separator
    } = config;

    const responsiveRules = _postcss.default.root();

    const finalRules = [];
    css.walkAtRules('responsive', atRule => {
      const nodes = atRule.nodes;
      responsiveRules.append(...(0, _cloneNodes.default)(nodes));
      atRule.before(nodes);
      atRule.remove();
    });

    _lodash.default.keys(screens).forEach(screen => {
      const mediaQuery = _postcss.default.atRule({
        name: 'media',
        params: (0, _buildMediaQuery.default)(screens[screen])
      });

      mediaQuery.append(_lodash.default.tap(responsiveRules.clone(), clonedRoot => {
        clonedRoot.walkRules(rule => {
          rule.selectors = _lodash.default.map(rule.selectors, selector => (0, _buildSelectorVariant.default)(selector, screen, separator, message => {
            throw rule.error(message);
          }));
        });
      }));
      finalRules.push(mediaQuery);
    });

    const hasScreenRules = finalRules.some(i => i.nodes.length !== 0);

    if (!hasScreenRules) {
      return;
    }

    let includesScreensExplicitly = false;
    css.walkAtRules('tailwind', atRule => {
      if (atRule.params === 'screens') {
        atRule.replaceWith(finalRules);
        includesScreensExplicitly = true;
      }
    });

    if (!includesScreensExplicitly) {
      css.append(finalRules);
      return;
    }
  };
}
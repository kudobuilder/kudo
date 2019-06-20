"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;

var _lodash = _interopRequireDefault(require("lodash"));

var _postcss = _interopRequireDefault(require("postcss"));

var _substituteTailwindAtRules = _interopRequireDefault(require("./lib/substituteTailwindAtRules"));

var _evaluateTailwindFunctions = _interopRequireDefault(require("./lib/evaluateTailwindFunctions"));

var _substituteVariantsAtRules = _interopRequireDefault(require("./lib/substituteVariantsAtRules"));

var _substituteResponsiveAtRules = _interopRequireDefault(require("./lib/substituteResponsiveAtRules"));

var _substituteScreenAtRules = _interopRequireDefault(require("./lib/substituteScreenAtRules"));

var _substituteClassApplyAtRules = _interopRequireDefault(require("./lib/substituteClassApplyAtRules"));

var _corePlugins = _interopRequireDefault(require("./corePlugins"));

var _processPlugins = _interopRequireDefault(require("./util/processPlugins"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _default(getConfig) {
  return function (css) {
    const config = getConfig();
    const processedPlugins = (0, _processPlugins.default)([...(0, _corePlugins.default)(config), ...config.plugins], config);
    return (0, _postcss.default)([(0, _substituteTailwindAtRules.default)(config, processedPlugins), (0, _evaluateTailwindFunctions.default)(config), (0, _substituteVariantsAtRules.default)(config, processedPlugins), (0, _substituteResponsiveAtRules.default)(config), (0, _substituteScreenAtRules.default)(config), (0, _substituteClassApplyAtRules.default)(config, processedPlugins.utilities)]).process(css, {
      from: _lodash.default.get(css, 'source.input.file')
    });
  };
}
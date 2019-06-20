"use strict";

Object.defineProperty(exports, "__esModule", {
  value: true
});
exports.default = _default;

var _lodash = _interopRequireDefault(require("lodash"));

var _flattenColorPalette = _interopRequireDefault(require("../util/flattenColorPalette"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function _default() {
  return function ({
    addUtilities,
    e,
    theme,
    variants
  }) {
    const utilities = _lodash.default.fromPairs(_lodash.default.map((0, _flattenColorPalette.default)(theme('stroke')), (value, modifier) => {
      return [`.${e(`stroke-${modifier}`)}`, {
        stroke: value
      }];
    }));

    addUtilities(utilities, variants('stroke'));
  };
}
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
    const colors = (0, _flattenColorPalette.default)(theme('borderColor'));

    const utilities = _lodash.default.fromPairs(_lodash.default.map(_lodash.default.omit(colors, 'default'), (value, modifier) => {
      return [`.${e(`border-${modifier}`)}`, {
        'border-color': value
      }];
    }));

    addUtilities(utilities, variants('borderColor'));
  };
}
"use strict";

var _fs = _interopRequireDefault(require("fs"));

var _postcss = _interopRequireDefault(require("postcss"));

var _ = _interopRequireDefault(require(".."));

var _cleanCss = _interopRequireDefault(require("clean-css"));

function _interopRequireDefault(obj) { return obj && obj.__esModule ? obj : { default: obj }; }

function buildDistFile(filename) {
  return new Promise((resolve, reject) => {
    console.log(`Processing ./${filename}.css...`);

    _fs.default.readFile(`./${filename}.css`, (err, css) => {
      if (err) throw err;
      return (0, _postcss.default)([(0, _.default)(), require('autoprefixer')]).process(css, {
        from: `./${filename}.css`,
        to: `./dist/${filename}.css`,
        map: {
          inline: false
        }
      }).then(result => {
        _fs.default.writeFileSync(`./dist/${filename}.css`, result.css);

        if (result.map) {
          _fs.default.writeFileSync(`./dist/${filename}.css.map`, result.map);
        }

        return result;
      }).then(result => {
        const minified = new _cleanCss.default().minify(result.css);

        _fs.default.writeFileSync(`./dist/${filename}.min.css`, minified.styles);
      }).then(resolve).catch(error => {
        console.log(error);
        reject();
      });
    });
  });
}

console.info('Building Tailwind!');
Promise.all([buildDistFile('base'), buildDistFile('components'), buildDistFile('utilities'), buildDistFile('tailwind')]).then(() => {
  console.log('Finished Building Tailwind!');
});
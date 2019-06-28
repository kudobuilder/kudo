class TailwindExtractor {
  static extract(content) {
    return content.match(/[A-z0-9-:\/]+/g)
  }
}

module.exports = {
  plugins: [
    require('postcss-import')({
      path: ["assets/css"],
    }),
    require('tailwindcss')('./assets/css/tailwind.config.js'),
    require('postcss-nesting')(),
    require('@fullhuman/postcss-purgecss')({
      content: ['layouts/**/*.html'],
      extractors: [
        {
          extractor: TailwindExtractor,
          extensions: ['html']
        }],
      fontFace: false,
      whitelist: ['class1', 'class2']
    }),
    require('autoprefixer')({
      grid: true,
      browsers: ['>1%']
    }),
  ]
}

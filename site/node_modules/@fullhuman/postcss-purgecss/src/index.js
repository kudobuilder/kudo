// @flow

import path from 'path'
import postcss from 'postcss'
import Purgecss from 'purgecss'

const CONFIG_FILENAME = 'purgecss.config.js'
const ERROR_CONFIG_FILE_LOADING = 'Error loading the config file'

const loadConfigFile = configFile => {
    const pathConfig =
        typeof configFile === 'undefined' ? CONFIG_FILENAME : configFile
    let options
    try {
        const t = path.resolve(process.cwd(), pathConfig)
        options = require(t)
    } catch (e) {
        throw new Error(ERROR_CONFIG_FILE_LOADING + e.message)
    }
    return options
}

export default postcss.plugin('postcss-plugin-purgecss', function(opts) {
    return function(root) {
        if (typeof opts === 'string' || typeof opts === 'undefined')
            opts = loadConfigFile(opts)

        if (!opts.css || !opts.css.length) {
            opts.css = ['__postcss_purgecss_placeholder']
        }

        const purgecss = new Purgecss(opts)
        purgecss.root = root

        // Get selectors from content files
        const { content, extractors } = purgecss.options

        const fileFormatContents = ((content.filter(
            o => typeof o === 'string'
        ): Array<any>): Array<string>)
        const rawFormatContents = ((content.filter(
            o => typeof o === 'object'
        ): Array<any>): Array<Purgecss.RawContent>)

        const cssFileSelectors = purgecss.extractFileSelector(
            fileFormatContents,
            extractors
        )
        const cssRawSelectors = purgecss.extractRawSelector(
            rawFormatContents,
            extractors
        )

        // Get css selectors and remove unused ones
        const cssSelectors = new Set([...cssFileSelectors, ...cssRawSelectors])

        // purge selectors
        purgecss.getSelectorsCss(cssSelectors)

        // purge keyframes
        if (purgecss.options.keyframes) purgecss.removeUnusedKeyframes()

        // purge font face
        if (purgecss.options.fontFace) purgecss.removeUnusedFontFaces()
    }
})

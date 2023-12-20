import { defineConfig, Plugin } from 'vite'
import { OutputChunk, OutputAsset } from "rollup"
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { PurgeCSS, UserDefinedOptions } from "purgecss";

export default defineConfig(() => {
  const html = process.env.HTML || "index.html"
  return {
    plugins: [svelte(), purgeCssPlugin(), singleFilePlugin()],
    build: {
      rollupOptions: {
        input: html,
      },
    },
    base: "./",
    server: {
      watch: { usePolling: true },
      port: 3000,
      proxy: { "^/api/.*": "http://localhost:3001" },
    },
  }
})

function purgeCssPlugin(): Plugin {
  return {
    name: 'vite:purgeCss',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const htmls = Object.keys(bundle).filter(key => key.endsWith('.html'));
      if (!htmls) {
        return
      }

      for (const html of htmls) {
        let filter = html.replace(".html", "")
        let re = new RegExp(`^assets/${filter}.*js$|^${filter}.*html$`)
        const jss = Object.keys(bundle).filter(key => re.test(key));
        const contents = jss.map(r => {
          const b = bundle[r] as any;
          if (r.endsWith('.js')) {
            return { raw: b.code, extension: "js" }
          }
          return { raw: b.source, extension: "html" }
        })
        re = new RegExp(`^assets/${filter}.*css$`)
        const cssNames = Object.keys(bundle).filter(key => re.test(key));
        if (!cssNames[0]) {
          continue
        }

        const options: UserDefinedOptions = {
          content: contents,
          css: [{ raw: (bundle[cssNames[0]] as any).source }],
          output: "dist/"
        }
        const purged = await new PurgeCSS().purge(options);
        (bundle[cssNames[0]] as any).source = purged[0].css;
      }
    }
  }
}

function singleFilePlugin(): Plugin {
  return {
    name: 'vite:singleFile',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const htmlNames = Object.keys(bundle).filter(key => key.endsWith('.html'));
      if (!htmlNames || htmlNames.length != 1) {
        console.log("必ず、1つのHTMLファイルを指定する必要があります。複数のHTMLファイルは指定できません。")
        return
      }

      const deleteTarget = [] as string[]
      const htmlAsset = bundle[htmlNames[0]] as OutputAsset
      let filter = htmlNames[0].replace(".html", "")
      let body = htmlAsset.source as string

      let re = new RegExp(`^assets/${filter}.*js$`)
      const jsNames = Object.keys(bundle).filter(key => re.test(key));

      for (const jsName of jsNames) {
        const target = `<script type="module" crossorigin src="./${jsName}"></script>`
        re = new RegExp(target)
        if (re.test(body)) {
          const jsChunk = bundle[jsName] as OutputChunk
          const replaced = `<script type="module" crossorigin>\n${jsChunk.code}\n</script>`
          const targets = body.split(target)
          body = targets[0] + replaced + targets[1]
          htmlAsset.source = body
          deleteTarget.push(jsName)
        }
      }
      re = new RegExp(`^assets/${filter}.*css$`)
      const cssNames = Object.keys(bundle).filter(key => re.test(key));

      for (const css of cssNames) {
        const target = `<link rel="stylesheet" crossorigin href="./${css}">`
        re = new RegExp(target)
        if (re.test(body)) {
          const replaced = `<style type="text/css">\n${(bundle[css] as any).source}\n</style>`
          body = body.replace(target, replaced);
          htmlAsset.source = body
          deleteTarget.push(css)
        }
      }
      for (const key of deleteTarget) {
        delete bundle[key]
      }
    }
  }
}
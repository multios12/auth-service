import { defineConfig, Plugin } from 'vite'
import { OutputChunk, OutputAsset, OutputBundle } from "rollup"
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { PurgeCSS, RawContent, UserDefinedOptions } from "purgecss"
import { resolve } from 'path'

export default defineConfig(() => {
  return {
    root: resolve(__dirname, 'src'),
    plugins: [svelte(), purgeCssPlugin(), singleFilePlugin2()],
    build: {
      rollupOptions: {
        input: {
          login: resolve(__dirname, 'src', 'login.html'),
        },
      },
      emptyOutDir: true,
      outDir: '../dist'
    },
    base: "./",
    server: {
      watch: { usePolling: true },
      port: 3000,
      proxy: { "^.*(auth|logout|start|info)$": "http://localhost:3001/auth/" },
    },
  }
})

function purgeCssPlugin2(): Plugin {
  return {
    name: 'vite:purgeCss',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const cssNames = Object.keys(bundle).filter(key => key.endsWith('.css'));
      for (const cssName of cssNames) {
        const content = createContent(cssName, bundle)
        const options: UserDefinedOptions = {
          content,
          css: [{ raw: (bundle[cssName] as any).source }],
          output: "dist/"
        }
        const purged = await new PurgeCSS().purge(options);
        (bundle[cssName] as any).source = purged[0].css;
      }
    }
  }
}

function createContent(cssName: string, bundle: OutputBundle): (string | RawContent<string>)[] {
  const contents: RawContent<string>[] = []

  for (const filename of Object.keys(bundle)) {
    const content = bundle[filename] as any
    const body = filename.endsWith('.js') ? content.code : content.source
    let re = new RegExp(cssName)
    if (re.test(body)) {
      contents.push({ raw: body, extension: filename.endsWith('.js') ? 'js' : 'html' })
    }
  }
  return contents
}

function singleFilePlugin2(): Plugin {
  return {
    name: 'vite:singleFile',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const htmlNames = Object.keys(bundle).filter(key => key.endsWith('.html'));

      const deleteTarget = [] as string[]
      for (const htmlName of htmlNames) {
        const htmlAsset = bundle[htmlName] as OutputAsset
        let body = htmlAsset.source as string

        let re = new RegExp(`.*js$|.*css$`)
        const names = Object.keys(bundle).filter(key => re.test(key))
        for (const name of names) {
          re = new RegExp(`<script type="module" crossorigin src="./${name}"></script>`
            + `|<link rel="modulepreload" crossorigin href="./${name}">`
            + `|<link rel="stylesheet" crossorigin href="./${name}">`)
          if (re.test(body)) {
            const replaced = name.endsWith('js') ? `<script type="module" crossorigin>\n${(bundle[name] as OutputChunk).code}\n    </script>` :
              `<style type="text/css">\n${(bundle[name] as any).source}\n    </style>`
            body = body.replace(re, replaced)
            deleteTarget.push(name)
          }
        }

        htmlAsset.source = body
      }

      for (const key of deleteTarget) {
        delete bundle[key]
      }
    }
  }
}

function purgeCssPlugin(): Plugin {
  return {
    name: 'vite:purgeCss',
    enforce: 'post',
    async generateBundle(_options, bundle) {
      const htmlNames = Object.keys(bundle).filter(key => key.endsWith('.html'));

      for (const html of htmlNames) {
        let filter = html.replace(".html", "")
        let re = new RegExp(`^assets/(${filter}|bulma).*js$|^${filter}.*html$`)
        const jss = Object.keys(bundle).filter(key => re.test(key));
        const contents = jss.map(r => {
          const b = bundle[r] as any;
          if (r.endsWith('.js')) {
            return { raw: b.code, extension: "js" }
          }
          return { raw: b.source, extension: "html" }
        })
        re = new RegExp(`^assets/(${filter}|bulma).*css$`)
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
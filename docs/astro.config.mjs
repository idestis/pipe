import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

const jsonLd = JSON.stringify({
  "@context": "https://schema.org",
  "@graph": [
    {
      "@type": "WebSite",
      name: "Pipe Docs",
      url: "https://docs.getpipe.dev",
      description:
        "Documentation for pipe — a minimal, dependency-free pipeline runner for the command line.",
      publisher: {
        "@type": "Organization",
        name: "Pipe",
        url: "https://getpipe.dev",
      },
    },
    {
      "@type": "SoftwareApplication",
      name: "pipe",
      url: "https://getpipe.dev",
      applicationCategory: "DeveloperApplication",
      operatingSystem: "Linux, macOS, Windows",
      offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
    },
  ],
});

export default defineConfig({
  site: "https://docs.getpipe.dev",
  integrations: [
    starlight({
      title: "pipe",
      logo: {
        src: "./src/assets/logo.png",
        alt: "pipe",
      },
      description:
        "A minimal, dependency-free pipeline runner for the command line.",
      social: {
        github: "https://github.com/getpipe-dev/pipe",
      },
      customCss: ["./src/styles/custom.css"],
      components: {
        Footer: "./src/components/Footer.astro",
      },
      head: [
        {
          tag: "meta",
          attrs: { name: "theme-color", content: "#0a0f1a" },
        },
        {
          tag: "link",
          attrs: {
            rel: "icon",
            type: "image/x-icon",
            href: "/favicon.ico",
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "icon",
            type: "image/png",
            sizes: "32x32",
            href: "/favicon-32x32.png",
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "icon",
            type: "image/png",
            sizes: "16x16",
            href: "/favicon-16x16.png",
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "apple-touch-icon",
            sizes: "180x180",
            href: "/apple-touch-icon.png",
          },
        },
        {
          tag: "link",
          attrs: { rel: "manifest", href: "/site.webmanifest" },
        },
        {
          tag: "meta",
          attrs: {
            property: "og:image",
            content: "https://docs.getpipe.dev/og-image.png",
          },
        },
        {
          tag: "meta",
          attrs: {
            property: "og:image:alt",
            content: "Pipe — pipeline runner for the command line",
          },
        },
        {
          tag: "script",
          attrs: { type: "application/ld+json" },
          content: jsonLd,
        },
      ],
      sidebar: [
        {
          label: "Get Started",
          items: [
            { slug: "getting-started/introduction" },
            { slug: "getting-started/installation" },
            { slug: "getting-started/quickstart" },
          ],
        },
        {
          label: "Guides",
          items: [
            { slug: "guides/writing-pipelines" },
            { slug: "guides/running-pipelines" },
            { slug: "guides/variables" },
            { slug: "guides/dependencies" },
            { slug: "guides/parallel-execution" },
            { slug: "guides/resuming-runs" },
            { slug: "guides/sensitive-data" },
            { slug: "guides/caching" },
            { slug: "guides/aliases" },
          ],
        },
        {
          label: "Hub",
          badge: { text: "Beta", variant: "caution" },
          items: [
            { slug: "hub/overview" },
            { slug: "hub/authentication" },
            { slug: "hub/push-pull" },
            { slug: "hub/tags" },
            { slug: "hub/converting-local" },
          ],
        },
        {
          label: "Reference",
          items: [
            { slug: "reference/cli" },
            {
              label: "Core Commands",
              items: [
                { slug: "reference/cli/run" },
                { slug: "reference/cli/init" },
                { slug: "reference/cli/list" },
                { slug: "reference/cli/lint" },
                { slug: "reference/cli/inspect" },
                { slug: "reference/cli/rm" },
                { slug: "reference/cli/cache" },
                { slug: "reference/cli/alias" },
              ],
            },
            {
              label: "Hub Commands",
              badge: { text: "Beta", variant: "caution" },
              items: [
                { slug: "reference/cli/login" },
                { slug: "reference/cli/logout" },
                { slug: "reference/cli/whoami" },
                { slug: "reference/cli/pull" },
                { slug: "reference/cli/push" },
                { slug: "reference/cli/mv" },
                { slug: "reference/cli/switch" },
                { slug: "reference/cli/tag" },
              ],
            },
            { slug: "reference/yaml-schema" },
            { slug: "reference/environment-variables" },
            { slug: "reference/directory-structure" },
          ],
        },
        {
          label: "Examples",
          autogenerate: { directory: "examples" },
        },
      ],
    }),
  ],
});

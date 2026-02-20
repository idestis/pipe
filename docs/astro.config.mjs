import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";

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

# Testkube docs

You can find the docs here: https://kubeshop.github.io/testkube/

## Edit the docs

If you're editing the docs, follow this workflow:

1. Install dependencies with `npm install`
2. Spin up local development with `npm run start`
3. Update the docs inside the `/docs` folder
4. Make sure to add the corresponding meta data on top of your markdown file if you want a specific label on the navigation or change the sort order:

```md
---
sidebar_position: 10
sidebar_label: cURL
---
```
5. You can preview the changes locally in your browser: http://localhost:3000/testkube/
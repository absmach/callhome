# Web Assets

This directory contains the web application's static assets (CSS and JavaScript).

## File Structure

- `style.css` - Source CSS file (clean, readable)
- `style.min.css` - Minified CSS (generated, do not edit)
- `app.js` - Source JavaScript file (clean, readable)
- `app.min.js` - Minified JavaScript (generated, do not edit)

## Development

During development, you can work with the clean source files:
- Edit `style.css` for styling changes
- Edit `app.js` for JavaScript functionality

The minified files are automatically generated during the build process and should not be edited manually.

## Building Minified Assets

### Local Development

To build minified assets locally:

```bash
# Install dependencies (first time only)
npm install

# Build minified CSS and JS
npm run build

# Or build individually
npm run build:css  # Minify CSS only
npm run build:js   # Minify JavaScript only

# Or use the Makefile
make build-assets
```

### Docker Build

The Docker build process automatically minifies assets during image creation. No manual intervention required.

```bash
make docker-image
```

## Production

The HTML template (`web/template/index.html`) references the minified versions:
- `/style.min.css`
- `/app.min.js`

These files must be generated before deploying or running the application in production.

### Go Template Data Injection

Since external JavaScript files are not processed by the Go template engine, server-side data is injected via a global variable in the HTML template:

```html
<script type="text/javascript">
  window.MAP_DATA = `{{.MapData}}`;
</script>
<script src="/app.min.js"></script>
```

The JavaScript code then reads from `window.MAP_DATA` instead of using template syntax directly.

## Tools Used

- **clean-css-cli**: CSS minification
- **terser**: JavaScript minification and compression

Both tools are configured in `package.json` and provide optimal compression while maintaining functionality.

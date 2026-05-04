const ModuleFederationPlugin = require("webpack/lib/container/ModuleFederationPlugin");
const HtmlWebpackPlugin = require("html-webpack-plugin");
const path = require("path");

module.exports = {
  entry: "./src/index.js",
  mode: "development",
  output: {
    filename: "main.js",
    publicPath: "auto",
    path: path.resolve(__dirname, "dist"),
  },
  devServer: {
    host: "0.0.0.0",
    port: 3000,
    allowedHosts: "all",
    historyApiFallback: true,
    static: path.resolve(__dirname, "dist"),
    proxy: [
      {
        context: ["/api/services/unifi"],
        target: "http://127.0.0.1:8081",
        changeOrigin: true,
        pathRewrite: {
          "^/api/services/unifi": "",
        },
      },
      {
        context: ["/ui/services/unifi"],
        target: "http://127.0.0.1:8081",
        changeOrigin: true,
        pathRewrite: {
          "^/ui/services/unifi": "/ui",
        },
      },
      {
        context: ["/api/services/cluster"],
        target: "http://127.0.0.1:8082",
        changeOrigin: true,
        pathRewrite: {
          "^/api/services/cluster": "",
        },
      },
      {
        context: ["/ui/services/cluster"],
        target: "http://127.0.0.1:8082",
        changeOrigin: true,
        pathRewrite: {
          "^/ui/services/cluster": "/ui",
        },
      },
      {
        context: ["/api/services/pfsense"],
        target: "http://127.0.0.1:8083",
        changeOrigin: true,
        pathRewrite: {
          "^/api/services/pfsense": "",
        },
      },
      {
        context: ["/ui/services/pfsense"],
        target: "http://127.0.0.1:8083",
        changeOrigin: true,
        pathRewrite: {
          "^/ui/services/pfsense": "/ui",
        },
      },
    ],
  },
  module: {
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /node_modules/,
        use: {
          loader: "babel-loader",
          options: {
            presets: ["@babel/preset-env", "@babel/preset-react"],
          },
        },
      },
      {
        test: /\.css$/,
        use: ["style-loader", "css-loader"],
      },
    ],
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: "./public/index.html",
    }),
    new ModuleFederationPlugin({
      name: "polaris",
      shared: {
        react: {
          singleton: true,
        },
        "react-dom": {
          singleton: true,
        },
      },
    }),
  ],
  resolve: {
    extensions: [".js", ".jsx"],
  },
};

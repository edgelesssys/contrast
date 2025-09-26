const { visit } = require("unist-util-visit");
const h = require("hastscript");

module.exports = function remarkFaq() {
  return (tree) => {
    visit(tree, (node) => {
      if (node.type === "containerDirective" && node.name === "faq") {
        // Take the first child paragraph’s text as the title
        let title = "FAQ Item";
        if (node.children.length > 0 && node.children[0].type === "paragraph") {
          title = node.children[0].children
            .map((c) => c.value || "")
            .join(" ")
            .trim();
          // remove that paragraph so it doesn’t render twice
          node.children = node.children.slice(1);
        }
        node.data = {
          hName: "FAQItem",
          hProperties: { title },
        };
      }
    });
  };
};

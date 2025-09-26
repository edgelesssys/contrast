import React, { useState } from "react";

export default function FAQItem({ title, children, isOpenDefault = false }) {
  const [isOpen, setIsOpen] = useState(isOpenDefault);

  return (
    <div className="faq-item">
      <h3
        className="faq-header"
        onClick={() => setIsOpen(!isOpen)}
      >
        {title}
      </h3>
      {isOpen && <div className="faq-body">{children}</div>}
    </div>
  );
}

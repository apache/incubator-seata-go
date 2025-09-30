/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { oneDark } from 'react-syntax-highlighter/dist/esm/styles/prism';
import './MarkdownRenderer.css';

interface MarkdownRendererProps {
  content: string;
  className?: string;
}

const MarkdownRenderer: React.FC<MarkdownRendererProps> = ({ content, className = '' }) => {
  return (
    <div className={`markdown-content ${className}`}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
        // 自定义代码块渲染
        code({ className, children, ...props }: any) {
          const match = /language-(\w+)/.exec(className || '');
          const inline = !className || !match;
          return !inline && match ? (
            <SyntaxHighlighter
              style={oneDark as any}
              language={match[1]}
              PreTag="div"
              {...props}
            >
              {String(children).replace(/\n$/, '')}
            </SyntaxHighlighter>
          ) : (
            <code className={className} {...props}>
              {children}
            </code>
          );
        },
        // 自定义表格样式
        table({ children }) {
          return (
            <div className="table-wrapper">
              <table>{children}</table>
            </div>
          );
        },
        // 自定义链接，添加安全属性
        a({ href, children }) {
          return (
            <a href={href} target="_blank" rel="noopener noreferrer">
              {children}
            </a>
          );
        },
        // 自定义列表样式
        ul({ children }) {
          return <ul className="markdown-list">{children}</ul>;
        },
        ol({ children }) {
          return <ol className="markdown-list">{children}</ol>;
        },
        // 自定义引用块样式
        blockquote({ children }) {
          return <blockquote className="markdown-blockquote">{children}</blockquote>;
        },
        // 自定义标题样式
        h1: ({ children }) => <h1 className="markdown-h1">{children}</h1>,
        h2: ({ children }) => <h2 className="markdown-h2">{children}</h2>,
        h3: ({ children }) => <h3 className="markdown-h3">{children}</h3>,
        h4: ({ children }) => <h4 className="markdown-h4">{children}</h4>,
        h5: ({ children }) => <h5 className="markdown-h5">{children}</h5>,
        h6: ({ children }) => <h6 className="markdown-h6">{children}</h6>,
      }}
    >
      {content}
    </ReactMarkdown>
    </div>
  );
};

export default MarkdownRenderer;
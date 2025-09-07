#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import re
import sys
from pathlib import Path


def contains_chinese(text):
    """检查文本是否包含中文字符"""
    chinese_pattern = re.compile(r'[\u4e00-\u9fff]+')
    return bool(chinese_pattern.search(text))


def extract_comments_from_line(line, file_extension):
    """根据文件类型提取注释内容"""
    comments = []
    line = line.strip()

    if file_extension in ['.py', '.sh', '.rb', '.pl', '.yaml', '.yml']:
        # Python, Shell, Ruby, Perl, YAML 使用 # 注释
        if '#' in line:
            comment_start = line.find('#')
            comment = line[comment_start:].strip()
            if comment and comment != '#':
                comments.append(comment)

    elif file_extension in ['.js', '.ts', '.java', '.c', '.cpp', '.h', '.hpp', '.cs', '.php', '.go', '.rs', '.swift',
                            '.kt']:
        # C风格语言使用 // 和 /* */ 注释
        # 处理单行注释 //
        if '//' in line:
            comment_start = line.find('//')
            comment = line[comment_start:].strip()
            if comment and comment != '//':
                comments.append(comment)

        # 处理多行注释 /* */ (简单处理，只检查单行中的)
        if '/*' in line and '*/' in line:
            start = line.find('/*')
            end = line.find('*/', start)
            if end != -1:
                comment = line[start:end + 2].strip()
                if comment and len(comment) > 4:  # 排除空的 /* */
                    comments.append(comment)
        elif '/*' in line or '*/' in line or line.strip().startswith('*'):
            # 多行注释的一部分
            comment = line.strip()
            if comment and comment not in ['/*', '*/']:
                comments.append(comment)

    elif file_extension in ['.html', '.xml', '.svg']:
        # HTML/XML 使用 <!-- --> 注释
        if '<!--' in line and '-->' in line:
            start = line.find('<!--')
            end = line.find('-->', start)
            if end != -1:
                comment = line[start:end + 3].strip()
                comments.append(comment)
        elif '<!--' in line or '-->' in line:
            # 多行注释的一部分
            comment = line.strip()
            if comment not in ['<!--', '-->']:
                comments.append(comment)

    elif file_extension in ['.css', '.scss', '.sass']:
        # CSS 使用 /* */ 注释
        if '/*' in line and '*/' in line:
            start = line.find('/*')
            end = line.find('*/', start)
            if end != -1:
                comment = line[start:end + 2].strip()
                comments.append(comment)
        elif '/*' in line or '*/' in line or (line.strip().startswith('*') and not line.strip().startswith('*/')):
            comment = line.strip()
            if comment not in ['/*', '*/']:
                comments.append(comment)

    elif file_extension in ['.sql']:
        # SQL 使用 -- 和 /* */ 注释
        if '--' in line:
            comment_start = line.find('--')
            comment = line[comment_start:].strip()
            if comment and comment != '--':
                comments.append(comment)
        # 处理SQL中的/* */注释
        if '/*' in line and '*/' in line:
            start = line.find('/*')
            end = line.find('*/', start)
            if end != -1:
                comment = line[start:end + 2].strip()
                if comment and len(comment) > 4:
                    comments.append(comment)

    return comments


def extract_non_comment_part(line, file_extension):
    """根据文件类型提取非注释部分（代码/字符串内容）"""
    non_comment = line.strip()
    if not non_comment:
        return ""

    # 处理 # 风格注释（Python, Shell, Ruby, Perl, YAML）
    if file_extension in ['.py', '.sh', '.rb', '.pl', '.yaml', '.yml']:
        # 简化处理：忽略字符串内的#（如需精确需解析字符串边界）
        if '#' in non_comment:
            non_comment = non_comment.split('#', 1)[0].strip()

    # 处理 C 风格注释（// 和 /* */）及 CSS, SQL
    elif file_extension in ['.js', '.ts', '.java', '.c', '.cpp', '.h', '.hpp',
                            '.cs', '.php', '.go', '.rs', '.swift', '.kt',
                            '.css', '.scss', '.sass', '.sql']:
        # 1. 先移除 /* */ 注释（单行内）
        non_comment = re.sub(r'/\*.*?\*/', '', non_comment, flags=re.DOTALL)
        # 2. 再移除单行注释（// 或 --）
        comment_prefix = '//' if file_extension != '.sql' else '--'
        if comment_prefix in non_comment:
            non_comment = non_comment.split(comment_prefix, 1)[0].strip()

    # 处理 HTML/XML 注释（<!-- -->）
    elif file_extension in ['.html', '.xml', '.svg']:
        non_comment = re.sub(r'<!--.*?-->', '', non_comment, flags=re.DOTALL)
        non_comment = non_comment.strip()

    # Markdown、纯文本、JSON 无特殊注释，全部视为非注释内容
    elif file_extension in ['.md', '.txt', '.json']:
        pass

    return non_comment


def should_skip_file(file_path):
    """判断是否应该跳过某个文件"""
    skip_patterns = [
        # 版本控制目录
        '.git/', '.svn/', '.hg/',
        # 构建目录
        'build/', 'dist/', 'target/', 'out/',
        # 依赖目录
        'node_modules/', 'vendor/', '.venv/', 'venv/', '__pycache__/',
        # IDE配置
        '.vscode/', '.idea/', '*.pyc', '*.pyo',
        # 日志和临时文件
        '*.log', '*.tmp', '*.temp',
        # 二进制文件
        '*.exe', '*.dll', '*.so', '*.dylib',
        # 图片文件
        '*.jpg', '*.jpeg', '*.png', '*.gif', '*.bmp', '*.svg', '*.ico',
        # 压缩文件
        '*.zip', '*.rar', '*.7z', '*.tar', '*.gz',
    ]

    file_str = str(file_path).replace('\\', '/')
    for pattern in skip_patterns:
        if pattern.endswith('/'):
            if f'/{pattern}' in f'/{file_str}' or file_str.startswith(pattern):
                return True
        elif '*' in pattern:
            if file_str.endswith(pattern.replace('*', '')):
                return True
        elif pattern in file_str:
            return True

    return False


def check_chinese_content(root_dir):
    """检查项目目录中的中文注释和代码中的中文内容"""
    root_path = Path(root_dir)
    if not root_path.exists():
        print(f"错误：目录 '{root_dir}' 不存在")
        return

    # 支持的文件扩展名
    supported_extensions = {
        '.py', '.js', '.ts', '.java', '.c', '.cpp', '.h', '.hpp',
        '.cs', '.php', '.go', '.rs', '.swift', '.kt', '.rb', '.pl',
        '.html', '.xml', '.css', '.scss', '.sass', '.sql', '.sh',
        '.yaml', '.yml', '.json', '.md', '.txt'
    }

    results = []
    total_files_checked = 0
    files_with_chinese = 0

    print(f"正在检查目录: {root_path.absolute()}")
    print("=" * 60)

    # 遍历所有文件
    for file_path in root_path.rglob('*'):
        if file_path.is_file():
            # 跳过不需要检查的文件
            if should_skip_file(file_path):
                continue

            file_extension = file_path.suffix.lower()

            # 只检查支持的文件类型
            if file_extension not in supported_extensions:
                continue

            total_files_checked += 1

            try:
                # 尝试以不同编码读取文件
                content = None
                encodings = ['utf-8', 'gbk', 'gb2312', 'latin1']

                for encoding in encodings:
                    try:
                        with open(file_path, 'r', encoding=encoding) as f:
                            content = f.read()
                        break
                    except UnicodeDecodeError:
                        continue

                if content is None:
                    print(f"警告：无法读取文件 {file_path}")
                    continue

                lines = content.splitlines()
                chinese_lines = []

                # 检查每一行的注释和非注释部分
                for line_num, line in enumerate(lines, 1):
                    # 1. 检查注释中的中文
                    comments = extract_comments_from_line(line, file_extension)
                    for comment in comments:
                        if contains_chinese(comment):
                            chinese_lines.append({
                                'line_num': line_num,
                                'content': line.strip(),
                                'part': '注释',
                                'detail': comment
                            })

                    # 2. 检查非注释部分（代码/字符串）中的中文
                    non_comment = extract_non_comment_part(line, file_extension)
                    if non_comment and contains_chinese(non_comment):
                        chinese_lines.append({
                            'line_num': line_num,
                            'content': line.strip(),
                            'part': '代码/字符串',
                            'detail': non_comment
                        })

                if chinese_lines:
                    files_with_chinese += 1
                    relative_path = file_path.relative_to(root_path)
                    results.append({
                        'file': str(relative_path),
                        'lines': chinese_lines
                    })

            except Exception as e:
                print(f"处理文件时出错 {file_path}: {e}")

    # 输出结果
    print(f"\n检查完成！")
    print(f"总共检查文件数: {total_files_checked}")
    print(f"包含中文内容的文件数: {files_with_chinese}")
    print("=" * 60)

    if results:
        print("\n发现以下文件包含中文内容：\n")

        for result in results:
            print(f"📁 文件: {result['file']}")
            print(f"   包含 {len(result['lines'])} 处中文内容")

            for line_info in result['lines']:
                # 显示行号、中文所在部分和完整行内容
                print(f"   第 {line_info['line_num']:4d} 行 [{line_info['part']}]: {line_info['content']}")
                # 显示具体包含中文的片段（避免与完整行重复）
                if line_info['detail'] != line_info['content']:
                    print(f"              中文片段: {line_info['detail']}")
            print()
    else:
        print("\n✅ 未发现任何中文内容")


def main():
    """主函数"""
    if len(sys.argv) > 1:
        root_dir = sys.argv[1]
    else:
        root_dir = input("请输入项目根目录路径（回车使用当前目录）: ").strip()
        if not root_dir:
            root_dir = "."

    check_chinese_content(root_dir)


if __name__ == "__main__":
    main()

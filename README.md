# goenkins-format

jenkins declarative pipeline formatter

## how to install
```
go get -u github.com/umaumax/goenkins-format
```

## how to use
```
cat xxx.groovy | goenkins-format
```

----

## FMI
### how to update codes
```
./build.sh

./test.sh
# or
./test.sh test/TODO_input
```

### NOTE
* 現在，字句解析のみで対応しているが，厳密には構文解析で対応する必要がある
* 通常，parserだとコメントはskipしても問題ない場合もあるが，formatterで構文解析でコードの出力処理の対応をする場合にはの場合にはskip不可
* githubで検索してみても，commentが挿入される可能性のある場所すべてに入れている
  * [Search · comment language:yacc]( https://github.com/search?q=comment+language%3Ayacc&type=Code )
    * [xserver/parser\.y at a8b31eff24d5a1f750b867cd99231bb3d9233217 · rib/xserver]( https://github.com/rib/xserver/blob/a8b31eff24d5a1f750b867cd99231bb3d9233217/hw/dmx/config/parser.y#L189 )
    * [ios\-toolchain\-based\-on\-clang\-for\-linux/pbxproj\.y at 05434f4c9f2e1c6d5f6834c0c90a0f4f5833335c · kydlo/ios\-toolchain\-based\-on\-clang\-for\-linux]( https://github.com/kydlo/ios-toolchain-based-on-clang-for-linux/blob/05434f4c9f2e1c6d5f6834c0c90a0f4f5833335c/iphonesdk-utils/xcbuild/libxcodeutils/pbxproj.y#L135 )
* `conflicts: xxx shift/reduce, xxx reduce/reduce`: 除去可能? そうだとしても，コストに見合うかどうか
  * これが，出現するケースとしては，下記のようなケースが原因であることが多い
    * 意図せずに空白がacceptされている状態
```
xxx: /* empty */
  | yyy
```
* 字句解析器の自動生成ライブラリ
  * [blynn/nex: Lexer for Go]( https://github.com/blynn/nex#nex-and-gos-yacc )

* [www\.hpcs\.cs\.tsukuba\.ac\.jp/~msato/lecture\-note/comp\-lecture/note5\.html]( http://www.hpcs.cs.tsukuba.ac.jp/~msato/lecture-note/comp-lecture/note5.html )

> ```
> seq:  item |  seq ',' term ;　/* left recursion */
> seq:  item | term ',' seq ;   /* right recursion */
> ```

yaccでは、right recursionでは、途中の状態をスタックにとっておく必要が あるため、なるべく、left recursionで書いておくべきである。

### links
#### jenkins
* about jenkins file
  * [Jenkinsfileの書き方 \- Qiita]( https://qiita.com/lufia/items/18cdb01f86a6d5040c60 )
* official pipeline syntax
  * [Pipeline Syntax]( https://jenkins.io/doc/book/pipeline/syntax/#compare )
* official linter tool
  * [Pipeline Development Tools]( https://jenkins.io/doc/book/pipeline/development/#linter )
* vscode linter plugin
  * [Validate your Jenkinsfile from within VS Code]( https://jenkins.io/blog/2018/11/07/Validate-Jenkinsfile/ )

#### yacc/lex
* [anko/parser\.go\.y at master · mattn/anko]( https://github.com/mattn/anko/blob/master/parser/parser.go.y )

#### 構文解析時のエラーハンドリング
* [goyaccで構文解析を行う \- Qiita]( https://qiita.com/k0kubun/items/1b641dfd186fe46feb65#yyparse%E3%81%AE%E5%BC%95%E6%95%B0 )

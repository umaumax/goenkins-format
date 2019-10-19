%{
package main
// import "fmt"
%}

%union {
  num int
  str string
  indent_level int
}

// NOTE: '\n'
%token NR
%token EOF
%token COMMENT

%token BOOL
%token NUMBER
%token STRING
// NOTE: 識別子
%token IDENT
%token DEF NEW
%token ANY NONE
%token SH ECHO
%token AGENT LABEL STAGE NODE DIR SCRIPT ENVIRONMENT
%token IMPORT
%token IF ELSE

// NOTE: low priority
// NOTE: no '\n'?
%left NR
// %left ','
%left OR
%left AND
%left '<' '>' EQ NE LE GE
%left '+' '-'
%left '*' '/' '%'
%right UNARY_OPERAND
%left '[' ']'
%left '(' ')'
%left '.'
// NOTE: high priority

%%

pipeline_stmts: /* blank */
  | pipeline_stmt pipeline_stmt_delimiter pipeline_stmts
  | pipeline_stmt_delimiter pipeline_stmts

groovy_stmts: /* blank */
  | groovy_stmt groovy_stmt_delimiter groovy_stmts
  | groovy_stmt_delimiter groovy_stmts

nop: /* blank */
   | nrs
   // | COMMENT

nrs: NR
  | nrs NR

groovy_stmt_delimiter: ';'
  | nrs

pipeline_stmt_delimiter: EOF
  | nrs

// NOTE: 文
pipeline_stmt: IMPORT package
  // NOTE: for other rules...
  | expr
  // | pipeline_block
  // NOTE: for other rules...
  | IDENT STRING
  // NOTE: for other rules...
  | IDENT expr
  | SH expr
  | ECHO expr
  | LABEL expr
  | AGENT ANY
  | AGENT NONE
  | AGENT pipeline_block
  // NOTE: for other rules...
  | IDENT pipeline_block
  | SCRIPT groovy_block
  // WARN: environment block rule is near script rule block
  | ENVIRONMENT expr
  | ENVIRONMENT groovy_block
  | STAGE '(' expr ')' pipeline_block
  | NODE '(' expr ')' pipeline_block
  | NODE pipeline_block
  | DIR '(' expr ')' pipeline_block
  // NOTE: for other rules...
  | IDENT '(' expr ')' pipeline_block
  // NOTE: for other rules...
  | IDENT '(' key_vals ')' pipeline_block

pipeline_block : '{' pipeline_stmts '}'

groovy_stmt: expr
  | groovy_block
  | DEF IDENT '=' expr
  // NOTE: for other rules...
  | IDENT IDENT '=' expr
  | expr '=' expr
  | IDENT groovy_block
  | ECHO expr
  | IF expr groovy_block
  | IF expr groovy_block ELSE groovy_block

groovy_block : '{' groovy_stmts '}'

package: IDENT
    | IDENT '.' package

exprs: /* blank */
    | expr
    | exprs ',' nop expr
    // | exprs ',' nrs expr
    // | exprs nop
    // | nrs exprs

key_vals: key_val
    | key_vals ',' nop key_val
    // | key_vals ',' nrs key_val
    // | key_vals nrs

key_val: IDENT ':' expr

// NOTE: 式
expr: primary
    // func call
    // | '[' exprs ']'
    | key_vals
    | '[' nop exprs nop ']'
    | '[' nop key_vals nop ']'
    // NOTE: duplicate rule but need for func()
    | IDENT '(' exprs ')'
    | expr '(' exprs ')'
    | expr '(' key_vals ')'
    | '(' key_vals ')'
    | expr '.' IDENT
    | NEW IDENT '(' exprs ')'
    // 演算記号
    | '-' expr %prec UNARY_OPERAND
    | expr '<' expr
    | expr '>' expr
    | expr '-' expr
    | expr '+' expr
    | expr '*' expr
    | expr '/' expr
    | expr '%' expr
    | expr EQ expr
    | expr NE expr
    | expr GE expr
    | expr LE expr
    | expr AND expr
    | expr OR expr

// NOTE: 項
primary : NUMBER
        | STRING
        | BOOL
        | IDENT
        | '(' expr ')'

%%

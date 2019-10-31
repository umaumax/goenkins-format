%{
package main
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
%token IF ELSE FOR IN TRY CATCH
%token INCREMENT DECREMENT
%token ARROW

// NOTE: low priority
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
  | pipeline_stmt
  | pipeline_stmt pipeline_stmt_delimiter pipeline_stmts
  | pipeline_stmt_delimiter pipeline_stmts

groovy_stmts: /* blank */
  | groovy_stmt
  | groovy_stmt groovy_stmt_delimiter groovy_stmts
  | groovy_stmt_delimiter groovy_stmts

nop: /* blank */
   | nop nrs
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
  // NOTE: for other rules...
  | DEF IDENT
  // NOTE: for other rules...
  | DEF IDENT '=' expr
  | DEF IDENT '(' nop exprs nop ')' pipeline_block
  | expr '=' expr
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
  | IDENT '(' nop key_vals nop ')' pipeline_block

pipeline_block : '{' pipeline_stmts '}'

groovy_stmt: expr
  | groovy_block
  | DEF IDENT
  | DEF IDENT '=' expr
  // NOTE: for other rules...
  | IDENT IDENT '=' expr
  | expr '=' expr
  | IDENT groovy_block
  | ECHO expr
  // NOTE: for other rules...
  | IDENT expr
  | SH expr
  | IF expr groovy_block
  | IF expr groovy_block ELSE groovy_block
  | FOR '(' IDENT IN expr ')' groovy_block
  | FOR '(' groovy_stmt ';' expr ';' expr ')' groovy_block
  | TRY groovy_block CATCH '(' IDENT IDENT ')' groovy_block
  // NOTE: for other rules...
  | DIR '(' expr ')' groovy_block
  | IDENT '(' expr ')' groovy_block
  // NOTE: lambda
  | exprs ARROW nop groovy_stmt
  | expr groovy_block

groovy_block : '{' groovy_stmts '}'

package: IDENT
    | '*'
    | IDENT '.' package

exprs: /* blank */
    | expr
    | exprs ',' nop expr

key_vals: key_val
    | key_vals ',' nop key_val

key_val: IDENT ':' expr
    // NOTE: for exception
    | SCRIPT ':' expr

// NOTE: 式
expr: primary
    | key_vals
    | '[' nop exprs nop ']'
    | '[' nop exprs ',' nop ']'
    | '[' nop key_vals nop ']'
    | '[' nop key_vals ',' nop ']'
    // NOTE: duplicate rule but need for func()
    | IDENT '(' nop exprs nop ')'
    // func call
    | expr '(' nop exprs nop ')'
    // NOTE: for exception
    | SH '(' nop key_vals nop ')'
    | expr '(' nop key_vals nop ')'
    | '(' nop key_vals nop ')'
    | expr '.' IDENT
    | NEW IDENT '(' nop exprs nop ')'
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
    | IDENT INCREMENT
    | IDENT DECREMENT

// NOTE: 項
primary : NUMBER
        | STRING
        | BOOL
        | IDENT
        | '(' expr ')'

%%

# Changelog

## [1.8.0](https://github.com/RomanAgaltsev/bigo/compare/v1.7.0...v1.8.0) (2026-07-11)


### Features

* **tripcount:** geometric loop bounds (R3/R4); heap corpus rework; ([#31](https://github.com/RomanAgaltsev/bigo/issues/31)) ([b19368a](https://github.com/RomanAgaltsev/bigo/commit/b19368a8118641143a5d5eadb4537ecaf0559947))

## [1.7.0](https://github.com/RomanAgaltsev/bigo/compare/v1.6.0...v1.7.0) (2026-07-11)


### Features

* **tripcount:** decreasing counted loops (R2); graduate insertion sort ([#29](https://github.com/RomanAgaltsev/bigo/issues/29)) ([b7119c7](https://github.com/RomanAgaltsev/bigo/commit/b7119c7eb0420f2b6b2c3bb9213f2e0cf659cb1e))

## [1.6.0](https://github.com/RomanAgaltsev/bigo/compare/v1.5.0...v1.6.0) (2026-07-11)


### Features

* shared trip-count facts and generalized increasing rule ([#27](https://github.com/RomanAgaltsev/bigo/issues/27)) ([bb9fbb9](https://github.com/RomanAgaltsev/bigo/commit/bb9fbb9397c05188889ff8c1b2fc7fd34fb1ee27))

## [1.5.0](https://github.com/RomanAgaltsev/bigo/compare/v1.4.0...v1.5.0) (2026-07-11)


### Features

* struct-field and receiver size variables (entry-stable paths) ([#25](https://github.com/RomanAgaltsev/bigo/issues/25)) ([b264413](https://github.com/RomanAgaltsev/bigo/commit/b264413c817110510f027547a8fff94c7038cb7e))

## [1.4.0](https://github.com/RomanAgaltsev/bigo/compare/v1.3.2...v1.4.0) (2026-07-11)


### Features

* metrics harness with committed coverage golden ([#23](https://github.com/RomanAgaltsev/bigo/issues/23)) ([edc4126](https://github.com/RomanAgaltsev/bigo/commit/edc4126b538239a690fe0b2df48496f519ce848a))

## [1.3.2](https://github.com/RomanAgaltsev/bigo/compare/v1.3.1...v1.3.2) (2026-07-10)


### Bug Fixes

* stop dropping budgets on multi-directive and bodyless declarations ([#19](https://github.com/RomanAgaltsev/bigo/issues/19)) ([324f73d](https://github.com/RomanAgaltsev/bigo/commit/324f73db62593443f3eae2277ff61cccd4549625))

## [1.3.1](https://github.com/RomanAgaltsev/bigo/compare/v1.3.0...v1.3.1) (2026-07-10)


### Bug Fixes

* reject variable offsets in loop conditions (wrong bound) ([#17](https://github.com/RomanAgaltsev/bigo/issues/17)) ([df25ed6](https://github.com/RomanAgaltsev/bigo/commit/df25ed6844070ba35013099baa9d4f18b8ae6235))

## [1.3.0](https://github.com/RomanAgaltsev/bigo/compare/v1.2.2...v1.3.0) (2026-07-10)


### Features

* honor //bigo:cost and //bigo:ignore, name unverifiable causes, scriptable -report ([#14](https://github.com/RomanAgaltsev/bigo/issues/14)) ([3b5fa61](https://github.com/RomanAgaltsev/bigo/commit/3b5fa618e11ca36e7069b953d59f0cd7c8e79950))

## [1.2.2](https://github.com/RomanAgaltsev/bigo/compare/v1.2.1...v1.2.2) (2026-07-10)


### Bug Fixes

* cap/len substitution and irreducible-CFG soundness ([#12](https://github.com/RomanAgaltsev/bigo/issues/12)) ([78b9929](https://github.com/RomanAgaltsev/bigo/commit/78b9929a9f6f013dc1af8c112b51a61e57f6fd68))

## [1.2.1](https://github.com/RomanAgaltsev/bigo/compare/v1.2.0...v1.2.1) (2026-07-10)


### Bug Fixes

* loop soundness, defer/go costs, directive error reporting ([#10](https://github.com/RomanAgaltsev/bigo/issues/10)) ([82859f5](https://github.com/RomanAgaltsev/bigo/commit/82859f58a576fb9743d8028e8c88e4456271c296))

## [1.2.0](https://github.com/RomanAgaltsev/bigo/compare/v1.1.0...v1.2.0) (2026-07-10)


### Features

* cost tables, interprocedural, plugin ([#8](https://github.com/RomanAgaltsev/bigo/issues/8)) ([bda0a8e](https://github.com/RomanAgaltsev/bigo/commit/bda0a8e064f53449162b3995bbfca99619cbfe54))

## [1.1.0](https://github.com/RomanAgaltsev/bigo/compare/v1.0.0...v1.1.0) (2026-07-09)


### Features

* mvp engine ([#6](https://github.com/RomanAgaltsev/bigo/issues/6)) ([a71c9f6](https://github.com/RomanAgaltsev/bigo/commit/a71c9f6cfb439e118c29077828a68c28d9648739))

## 1.0.0 (2026-07-09)


### Features

* **annotation:** add directive Parse with verbs ([767180f](https://github.com/RomanAgaltsev/bigo/commit/767180f3faa8d32393ac8bb9296d05155443981f))
* **annotation:** add O() expression parser ([41bfefb](https://github.com/RomanAgaltsev/bigo/commit/41bfefbd7d244867cd360b0efea4e1a0f7e243da))
* **annotation:** add where-clause bindings and Parse fuzz test ([ccd8d56](https://github.com/RomanAgaltsev/bigo/commit/ccd8d560570a1ec4c695a75e154b85170b496aea))
* **annotation:** add where-clause bindings and Parse fuzz test ([13abeb5](https://github.com/RomanAgaltsev/bigo/commit/13abeb5f1d1b2c7df51371f79aae93eab3e8771a))
* **bound:** add asymptotic domination on monomials ([ef278a1](https://github.com/RomanAgaltsev/bigo/commit/ef278a1a59ea77ae3f4ace872274507f6de5baec))
* **bound:** add Bound antichain with top and join ([5321ce4](https://github.com/RomanAgaltsev/bigo/commit/5321ce47a2b9baa3b9d887b35739abf5e60a2a82))
* **bound:** add Bound.Mul loop operation ([f0426a6](https://github.com/RomanAgaltsev/bigo/commit/f0426a66c2ef8ad47619ed1e5ca647cd4234143c))
* **bound:** add poly-log Monomial type ([a4bc25e](https://github.com/RomanAgaltsev/bigo/commit/a4bc25e5f920fb0499225935531406fc13a0073c))
* **bound:** add three-valued budget verdict Check ([4e3227c](https://github.com/RomanAgaltsev/bigo/commit/4e3227cb8260c8ffc49121932646e4c2da48d442))
* foundations ([77acce9](https://github.com/RomanAgaltsev/bigo/commit/77acce9cc9282dc8fa6b2a33f97cffb2d3696e67))

INSERT INTO users (username, password) VALUES (
	'tester',
	crypt('secretkey', gen_salt('bf'))
);

INSERT INTO variants (name) VALUES (
	'История вычислительной техники'
);

INSERT INTO questions (variant_id, question) VALUES (
	1,
	'Основатель движения за свободное ПО.'
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	1,
	'Линус Торвальдс',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	1,
	'Деннис Ритчи',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	1,
	'Ричард Столлман',
	true
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	1,
	'Эрик Реймонд',
	false
);

INSERT INTO questions (variant_id, question) VALUES (
	1,
	'Автор идеи пайпов в командной оболочке Unix.'
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	2,
	'Кен Томпсон',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	2,
	'Брайан Керниган',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	2,
	'Деннис Ритчи',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	2,
	'Дуг Макилрой',
	true
);

INSERT INTO questions (variant_id, question) VALUES (
	1,
	'Автор языка Lisp.'
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	3,
	'Джон Маккарти',
	true
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	3,
	'Билл Госпер',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	3,
	'Том Найт',
	false
);

INSERT INTO answers_variants (question_id, answer, correct) VALUES (
	3,
	'Марвин Минский',
	false
);
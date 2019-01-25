-- add a trailing `/` when needed using POSIX regex to detect records to update
update cluster set url = concat(url, '/') where url !~ '.*\/$';
update cluster set console_url = concat(console_url, '/') where console_url !~ '.*\/$';
update cluster set metrics_url = concat(metrics_url, '/') where metrics_url !~ '.*\/$';
update cluster set logging_url = concat(logging_url, '/') where logging_url !~ '.*\/$';
update cluster set logging_url = concat(logging_url, '/') where logging_url !~ '.*\/$';

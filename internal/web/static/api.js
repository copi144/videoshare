// VideoShare API helper
(function() {
  'use strict';

  window.VS = window.VS || {};

  // JSON request helper
  async function jsonRequest(method, url, body) {
    const headers = {
      'Accept': 'application/json',
    };
    const opts = { method, headers };
    if (body !== undefined && method !== 'GET') {
      headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const res = await fetch(url, opts);
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
  }

  // Form data request helper (for file uploads)
  async function formRequest(method, url, formData) {
    const headers = {
      'Accept': 'application/json',
    };
    const opts = { method, headers, body: formData };
    const res = await fetch(url, opts);
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
  }

  // Hijack a form to submit via JSON
  function hijackForm(formId, apiUrl, method, successCallback) {
    const form = document.getElementById(formId);
    if (!form) return;
    form.addEventListener('submit', async function(e) {
      e.preventDefault();
      const errorEl = form.querySelector('.error-message');
      const submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) submitBtn.disabled = true;

      try {
        const formData = new FormData(form);
        const body = {};
        formData.forEach((value, key) => {
          body[key] = value;
        });
        const data = await jsonRequest(method, apiUrl, body);
        if (successCallback) {
          successCallback(data);
        } else if (data.redirect) {
          window.location.href = data.redirect;
        } else {
          window.location.reload();
        }
      } catch (err) {
        if (errorEl) {
          errorEl.textContent = err.message;
          errorEl.style.display = 'block';
        } else {
          alert('Error: ' + err.message);
        }
      } finally {
        if (submitBtn) submitBtn.disabled = false;
      }
    });
  }

  // Hijack a form for file upload (multipart/form-data)
  function hijackUploadForm(formId, apiUrl) {
    const form = document.getElementById(formId);
    if (!form) return;
    form.addEventListener('submit', async function(e) {
      e.preventDefault();
      const errorEl = form.querySelector('.error-message');
      const submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) submitBtn.disabled = true;

      try {
        const formData = new FormData(form);
        const data = await formRequest('POST', apiUrl, formData);
        if (data.redirect) {
          window.location.href = data.redirect;
        } else {
          window.location.reload();
        }
      } catch (err) {
        if (errorEl) {
          errorEl.textContent = err.message;
          errorEl.style.display = 'block';
        } else {
          alert('Error: ' + err.message);
        }
      } finally {
        if (submitBtn) submitBtn.disabled = false;
      }
    });
  }

  // Delete helper (click handler)
  function setupDeleteButton(btnId, apiUrl, confirmMsg) {
    const btn = document.getElementById(btnId);
    if (!btn) return;
    btn.addEventListener('click', async function(e) {
      e.preventDefault();
      if (confirmMsg && !confirm(confirmMsg)) return;
      const errorEl = document.getElementById('error-message');
      try {
        await jsonRequest('DELETE', apiUrl);
        window.location.reload();
      } catch (err) {
        if (errorEl) {
          errorEl.textContent = err.message;
          errorEl.style.display = 'block';
        }
      }
    });
  }

  // Checkbox array submission
  function setupCheckboxForm(formId, apiUrl, method) {
    const form = document.getElementById(formId);
    if (!form) return;
    form.addEventListener('submit', async function(e) {
      e.preventDefault();
      const errorEl = form.querySelector('.error-message');
      const submitBtn = form.querySelector('button[type="submit"]');
      if (submitBtn) submitBtn.disabled = true;

      try {
        const formData = new FormData(form);
        const body = {};
        // Handle array values (checkboxes)
        const userIds = [];
        formData.forEach((value, key) => {
          if (key === 'user_ids') {
            userIds.push(value);
          } else {
            body[key] = value;
          }
        });
        if (userIds.length > 0) body.user_ids = userIds;
        const data = await jsonRequest(method || 'POST', apiUrl, body);
        if (data.redirect) {
          window.location.href = data.redirect;
        } else {
          window.location.reload();
        }
      } catch (err) {
        if (errorEl) {
          errorEl.textContent = err.message;
          errorEl.style.display = 'block';
        }
      } finally {
        if (submitBtn) submitBtn.disabled = false;
      }
    });
  }

  // Export helpers
  window.VS.jsonRequest = jsonRequest;
  window.VS.formRequest = formRequest;
  window.VS.hijackForm = hijackForm;
  window.VS.hijackUploadForm = hijackUploadForm;
  window.VS.setupDeleteButton = setupDeleteButton;
  window.VS.setupCheckboxForm = setupCheckboxForm;
})();
